package things

import (
	"encoding/json"
	"fmt"
	"net/http"
	"regexp"
	"strings"
	"sync"
	"errors"

	"github.com/gorilla/mux"
	log "github.com/sirupsen/logrus"
	gouuid "github.com/satori/go.uuid"
)

type RequestHandler struct {
	ThingsDriver          Driver
	CacheControllerHeader string
}

const validUUID = "([0-9a-fA-F]{8}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{12})$"

// MethodNotAllowedHandler handles 405
func (rh *RequestHandler) MethodNotAllowedHandler(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusMethodNotAllowed)
	return
}

// GetThing handler directly returns the concept/thing if it's a canonical
// or provides redirect URL via Location http header within the response.
func (rh *RequestHandler) GetThing(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	uuid := vars["uuid"]

	relationships := r.URL.Query()["showRelationship"]

	w.Header().Set("Content-Type", "application/json; charset=UTF-8")
	if uuid == "" {
		http.Error(w, "uuid required", http.StatusBadRequest)
		return
	}

	if err := validateUUID(uuid); err != nil {
		http.Error(w, "invalid/malformed uuid", http.StatusBadRequest)
		return
	}

	thng, found, err := rh.ThingsDriver.read(uuid, relationships)
	if err != nil {
		w.WriteHeader(http.StatusServiceUnavailable)
		msg := fmt.Sprintf(`{"message":"Error getting thing with uuid %s, err=%s"}`, uuid, err.Error())
		w.Write([]byte(msg))
		return
	}
	if !found {
		w.WriteHeader(http.StatusNotFound)
		msg := fmt.Sprintf(`{"message":"No thing found with uuid %s."}`, uuid)
		w.Write([]byte(msg))
		return
	}

	//if the request was not made for the canonical, but an alternate uuid: redirect
	if !strings.Contains(thng.ID, uuid) {
		validRegexp := regexp.MustCompile(validUUID)
		canonicalUUID := validRegexp.FindString(thng.ID)
		redirectURL := strings.Replace(r.URL.String(), uuid, canonicalUUID, 1)
		w.Header().Set("Location", redirectURL)
		w.WriteHeader(http.StatusMovedPermanently)
		return
	}

	w.Header().Set("Cache-Control", rh.CacheControllerHeader)
	w.WriteHeader(http.StatusOK)

	if err = json.NewEncoder(w).Encode(thng); err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		msg := fmt.Sprintf(`{"message":"Error parsing thing with uuid %s, err=%s"}`, uuid, err.Error())
		w.Write([]byte(msg))
	}
}

// GetThings handler provides a batch like functionality, quite similar to single get endpoint.
// Implementation schedules a new go routine for every requested "thing" uuid, wait for the results and returns
// the aggregated results to caller.
//
// Non canonical uuid handling:
//
// 	Implementation slightly deviates from the single get endpoint for non canonical uuids.
// 	It tries to resolve the canonical uuid/node itself instead of providing a reference url but strictly stops
// 	if indirection dept is more than one level.
//
// Error handling:
//
// 	In case of any error for any given uuid, implementation immediately returns with the error without waiting for the
// 	in-flight queries to be finished.
//
// Response structure:
//
// 	Instead of returning an array of things, we are returning/serializing a map of ["things":{[uuid:Thing]}]. Reason behind this
// 	is simply to provide a convenient way to the caller for making the correlation between requested uuids with respect to
// 	found things. Since we are handling the resolution of non canonical uuids, returned thing payloads may not have the same
// 	requested/associated uuid.
func (rh *RequestHandler) GetThings(w http.ResponseWriter, r *http.Request) {

	queryParams := r.URL.Query()
	relationships := queryParams["showRelationship"]
	uuids := queryParams["uuid"]

	w.Header().Set("Content-Type", "application/json; charset=UTF-8")

	if len(uuids) == 0 {
		w.WriteHeader(http.StatusBadRequest)
		w.Write([]byte(`{"message":"at least one uuid query param should be provided for batch operations"}`))
		return
	}

	if err := validateUUID(uuids...); err != nil {
		w.WriteHeader(http.StatusBadRequest)
		w.Write([]byte(fmt.Sprintf(`{"message":"%v"}`, err)))
		return
	}

	var wg sync.WaitGroup
	uctCh := make(chan *uuidConceptTuple)
	errCh := make(chan *uuidErrorTuple)

	// fill up the sync bucket
	wg.Add(len(uuids))

	// start getting things
	for _, uuid := range uuids {
		go rh.getChanneledThing(uuid, relationships, uctCh, errCh, &wg)
	}

	// start watching the sync bucket and close the channel
	go closeOnDone(uctCh, &wg)

	// synchronize/wait for the results
	things, err := aggregateChanneledThings(uctCh, errCh)

	if err != nil {
		w.WriteHeader(http.StatusServiceUnavailable)
		msg := fmt.Sprintf(`{"message":"Error getting thing with uuid %s, err=%s"}`, err.uuid, err.err.Error())
		w.Write([]byte(msg))
		return
	}

	w.Header().Set("Cache-Control", rh.CacheControllerHeader)
	w.WriteHeader(http.StatusOK)

	result := make(map[string]map[string]Concept)
	result["things"] = things

	if err := json.NewEncoder(w).Encode(result); err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		msg := fmt.Sprintf(`{"message":"Error marshalling the result %s, err=%s"}`, result, err.Error())
		w.Write([]byte(msg))
	}
}

func (rh *RequestHandler) getChanneledThing(uuid string, relationships []string, uctCh chan *uuidConceptTuple,
	errCh chan *uuidErrorTuple, wg *sync.WaitGroup) {

	defer wg.Done()
	thing, found, err := rh.ThingsDriver.read(uuid, relationships)

	if err != nil {
		errCh <- &uuidErrorTuple{uuid, err}
		return
	}

	if !found {
		return
	}

	if !strings.Contains(thing.ID, uuid) {
		validRegexp := regexp.MustCompile(validUUID)

		canonicalUUID := validRegexp.FindString(thing.ID)
		thing, found, err = rh.ThingsDriver.read(canonicalUUID, relationships)

		if err != nil {
			errCh <- &uuidErrorTuple{uuid, err}
			return
		}

		if !found {
			log.Error("Referenced canonical uuid : %s is missing in graph store for %s, possible data inconsistency",
				canonicalUUID, uuid)
			return
		}

		if !strings.Contains(thing.ID, canonicalUUID) {
			// there should be one level of indirection to the canonical node
			log.Warn("Multiple level of indirection to canonical node for uuid: %s, giving up traversing", uuid)
			return
		}
	}

	uctCh <- &uuidConceptTuple{uuid, thing}
}

func aggregateChanneledThings(uctCh chan *uuidConceptTuple, errCh chan *uuidErrorTuple) (map[string]Concept, *uuidErrorTuple) {
	things := make(map[string]Concept)

infiniteUntilClosed:
	for {
		select {
		case tuple, open := <-uctCh:
			if !open {
				break infiniteUntilClosed
			}
			things[tuple.uuid] = tuple.concept
		case err := <-errCh:
			return nil, err
		}
	}
	return things, nil
}

func closeOnDone(uctCh chan *uuidConceptTuple, wg *sync.WaitGroup) {
	wg.Wait()
	close(uctCh)
}

type uuidConceptTuple struct {
	uuid    string
	concept Concept
}

type uuidErrorTuple struct {
	uuid string
	err  error
}

func validateUUID(uuids ...string) error {
	for _, uuid := range uuids {
		_, err := gouuid.FromString(uuid)
		if err != nil {
			return errors.New(fmt.Sprintf("Invalid uuid: %s, err: %v", uuid, err))
		}
	}
	return nil
}