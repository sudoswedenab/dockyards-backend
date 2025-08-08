package handlers_test

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"path"
	"testing"

	"github.com/sudoswedenab/dockyards-api/pkg/types"
	dockyardsv1 "github.com/sudoswedenab/dockyards-backend/api/v1alpha3"
	"golang.org/x/crypto/bcrypt"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

func TestUserPassword_Update(t *testing.T) {
	if os.Getenv("KUBEBUILDER_ASSETS") == "" {
		t.Skip("no kubebuilder assets configured")
	}

	c := testEnvironment.GetClient()

	t.Run("test update password", func(t *testing.T) {
		hash, err := bcrypt.GenerateFromPassword([]byte("testing"), bcrypt.DefaultCost)
		if err != nil {
			t.Fatal(err)
		}

		user := dockyardsv1.User{
			ObjectMeta: metav1.ObjectMeta{
				GenerateName: "dockyards-",
			},
			Spec: dockyardsv1.UserSpec{
				Password: string(hash),
			},
		}

		err = c.Create(ctx, &user)
		if err != nil {
			t.Fatal(err)
		}

		userToken := MustSignToken(t, user.Name)

		passwordOptions := types.PasswordOptions{
			OldPassword: "testing",
			NewPassword: "update",
		}

		b, err := json.Marshal(&passwordOptions)
		if err != nil {
			t.Fatal(err)
		}

		u := url.URL{
			Path: path.Join("/v1/users", user.Name, "password"),
		}

		w := httptest.NewRecorder()
		r := httptest.NewRequest(http.MethodPost, u.Path, bytes.NewBuffer(b))

		r.Header.Add("Authorization", "Bearer "+userToken)

		mux.ServeHTTP(w, r)

		statusCode := w.Result().StatusCode
		if statusCode != http.StatusAccepted {
			t.Fatalf("expected status code %d, got %d", http.StatusAccepted, statusCode)
		}

		var actual dockyardsv1.User
		err = c.Get(ctx, client.ObjectKeyFromObject(&user), &actual)
		if err != nil {
			t.Fatal(err)
		}

		err = bcrypt.CompareHashAndPassword([]byte(actual.Spec.Password), []byte("update"))
		if err != nil {
			t.Fatal(err)
		}
	})

	t.Run("test incorrect old password", func(t *testing.T) {
		hash, err := bcrypt.GenerateFromPassword([]byte("old"), bcrypt.DefaultCost)
		if err != nil {
			t.Fatal(err)
		}

		user := dockyardsv1.User{
			ObjectMeta: metav1.ObjectMeta{
				GenerateName: "dockyards-",
			},
			Spec: dockyardsv1.UserSpec{
				Password: string(hash),
			},
		}

		err = c.Create(ctx, &user)
		if err != nil {
			t.Fatal(err)
		}

		userToken := MustSignToken(t, user.Name)

		passwordOptions := types.PasswordOptions{
			OldPassword: "testing",
			NewPassword: "update",
		}

		b, err := json.Marshal(&passwordOptions)
		if err != nil {
			t.Fatal(err)
		}

		u := url.URL{
			Path: path.Join("/v1/users", user.Name, "password"),
		}

		w := httptest.NewRecorder()
		r := httptest.NewRequest(http.MethodPost, u.Path, bytes.NewBuffer(b))

		r.Header.Add("Authorization", "Bearer "+userToken)

		mux.ServeHTTP(w, r)

		statusCode := w.Result().StatusCode
		if statusCode != http.StatusUnauthorized {
			t.Fatalf("expected status code %d, got %d", http.StatusUnauthorized, statusCode)
		}
	})
}
