package telegram

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestGetUpdatesMovesOffset(t *testing.T) {
	offsets := make(chan int, 2)
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var params struct {
			Offset int `json:"offset"`
		}
		json.NewDecoder(r.Body).Decode(&params)
		offsets <- params.Offset
		fmt.Fprint(w, `{"ok":true,"result":[{"update_id":5},{"update_id":7}]}`)
	}))
	defer srv.Close()

	c, err := New("token", srv.Client())
	if err != nil {
		t.Fatal(err)
	}
	c.url = srv.URL

	for range 2 {
		if _, err := c.GetUpdates(context.Background()); err != nil {
			t.Fatal(err)
		}
	}

	if first, second := <-offsets, <-offsets; first != 0 || second != 8 {
		t.Errorf("offsets = %d, %d, want 0, 8", first, second)
	}
}

func TestSendMessageError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusForbidden)
		fmt.Fprint(w, `{"ok":false,"error_code":403,"description":"Forbidden: bot was blocked by the user"}`)
	}))
	defer srv.Close()

	c, err := New("token", srv.Client())
	if err != nil {
		t.Fatal(err)
	}
	c.url = srv.URL

	err = c.SendMessage(context.Background(), 1, "hi")
	if err == nil || !strings.Contains(err.Error(), "blocked") {
		t.Errorf("error = %v, want telegram description", err)
	}
}
