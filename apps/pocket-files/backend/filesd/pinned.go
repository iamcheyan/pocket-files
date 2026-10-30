package main

import (
	"encoding/json"
	"net/http"
	"strings"
)

// Pinned ("custom") locations are user shortcuts to a folder inside a storage
// provider. They store provider-relative URIs only, so the UI never learns an
// absolute path and a pin can never point outside a sandboxed root.
type pinnedLocation struct {
	ID        int64  `json:"id"`
	Name      string `json:"name"`
	StorageID string `json:"storage_id"`
	URI       string `json:"uri"`
}

func (b *Backend) listPinned() []pinnedLocation {
	rows, err := b.db.Query(`SELECT id,name,storage_id,uri FROM custom_locations ORDER BY id`)
	if err != nil {
		return []pinnedLocation{}
	}
	defer rows.Close()
	out := []pinnedLocation{}
	for rows.Next() {
		var p pinnedLocation
		if rows.Scan(&p.ID, &p.Name, &p.StorageID, &p.URI) == nil {
			out = append(out, p)
		}
	}
	return out
}

func (b *Backend) addPinned(name, storageID, uri string) (pinnedLocation, error) {
	if strings.TrimSpace(name) == "" {
		return pinnedLocation{}, errInvalidPin("name is required")
	}
	if _, ok := b.getDevice(storageID); !ok {
		return pinnedLocation{}, errInvalidPin("unknown storage")
	}
	clean := strings.Trim(strings.TrimSpace(uri), "/")
	// The target must resolve inside the provider root; a pin to a missing
	// directory is rejected so the sidebar cannot accumulate dead links. A pin
	// to the storage root itself (empty URI) is allowed.
	if clean != "" {
		if _, err := b.p.Resolve(storageID, clean, false); err != nil {
			return pinnedLocation{}, errInvalidPin("location does not exist inside this storage")
		}
	}
	res, err := b.db.Exec(`INSERT INTO custom_locations(name,storage_id,uri) VALUES(?,?,?)`, strings.TrimSpace(name), storageID, clean)
	if err != nil {
		return pinnedLocation{}, errInvalidPin("unable to save location")
	}
	id, _ := res.LastInsertId()
	return pinnedLocation{ID: id, Name: strings.TrimSpace(name), StorageID: storageID, URI: clean}, nil
}

func (b *Backend) deletePinned(id int64) error {
	res, err := b.db.Exec(`DELETE FROM custom_locations WHERE id=?`, id)
	if err != nil {
		return err
	}
	if affected, _ := res.RowsAffected(); affected == 0 {
		return errInvalidPin("location not found")
	}
	return nil
}

type errInvalidPin string

func (e errInvalidPin) Error() string { return string(e) }

// handlePinnedLocations serves GET/POST /api/locations and DELETE /api/locations/{id}.
func (s *Server) handlePinnedLocations(w http.ResponseWriter, r *http.Request, parts []string) {
	switch {
	case len(parts) == 2 && r.Method == http.MethodGet:
		s.json(w, http.StatusOK, map[string]any{"items": s.be.listPinned()})
	case len(parts) == 2 && r.Method == http.MethodPost:
		var body struct {
			Name      string `json:"name"`
			StorageID string `json:"storage_id"`
			URI       string `json:"uri"`
		}
		if json.NewDecoder(r.Body).Decode(&body) != nil {
			s.error(w, http.StatusBadRequest, "invalid_json", "invalid request")
			return
		}
		created, err := s.be.addPinned(body.Name, body.StorageID, body.URI)
		if err != nil {
			s.error(w, http.StatusBadRequest, "invalid_location", err.Error())
			return
		}
		s.json(w, http.StatusCreated, created)
	case len(parts) == 3 && r.Method == http.MethodDelete:
		id, err := parseInt64(parts[2])
		if err != nil {
			s.error(w, http.StatusBadRequest, "invalid_id", "location id must be numeric")
			return
		}
		if err = s.be.deletePinned(id); err != nil {
			s.error(w, http.StatusNotFound, "not_found", err.Error())
			return
		}
		s.json(w, http.StatusOK, map[string]any{"status": "ok", "id": id})
	default:
		s.error(w, http.StatusMethodNotAllowed, "method_not_allowed", "use GET, POST or DELETE")
	}
}

func parseInt64(value string) (int64, error) {
	var n int64
	if value == "" {
		return 0, errInvalidPin("empty id")
	}
	for _, r := range value {
		if r < '0' || r > '9' {
			return 0, errInvalidPin("non-numeric id")
		}
		n = n*10 + int64(r-'0')
	}
	return n, nil
}
