package user

import (
	"bytes"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"path"
	"testing"
	"time"

	"bitbucket.org/sudosweden/backend/api/v1/model"
	"github.com/gin-gonic/gin"
	"github.com/glebarez/sqlite"
	"gorm.io/gorm"
)

func TestFindAllUsers(t *testing.T) {
	gin.SetMode(gin.ReleaseMode)

	tt := []struct {
		name  string
		users []model.User
	}{
		{
			name: "test single user",
			users: []model.User{
				{
					ID:        1,
					Name:      "test",
					CreatedAt: time.Now(),
					UpdatedAt: time.Now(),
				},
			},
		},
		{
			name: "test multiple users",
			users: []model.User{
				{
					ID:    1,
					Name:  "user1",
					Email: "user1@dockyards.io",
				},
				{
					ID:    2,
					Name:  "user2",
					Email: "user2@dockyards.io",
				},
				{
					ID:    3,
					Name:  "user3",
					Email: "user3@dockyards.io",
				},
			},
		},
	}

	type response struct {
		Users []model.User `json:"user"`
	}

	for _, tc := range tt {
		t.Run(tc.name, func(t *testing.T) {
			db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
			if err != nil {
				t.Fatalf("unexpected error creating db: %s", err)
			}
			db.AutoMigrate(&model.User{})

			for _, user := range tc.users {
				tx := db.Create(&user)
				if tx.Error != nil {
					t.Fatalf("unxepected error preparing test database: %s", err)
				}
			}

			h := handler{
				db: db,
			}

			r := gin.New()
			r.GET("/test", h.FindAllUsers)

			w := httptest.NewRecorder()
			req, err := http.NewRequest(http.MethodGet, "/test", nil)
			if err != nil {
				t.Fatalf("unexpected error preparing request: %s", err)
			}

			r.ServeHTTP(w, req)

			if w.Code != http.StatusOK {
				t.Errorf("expected code %d, got %d", http.StatusOK, w.Code)
			}

			body, err := io.ReadAll(w.Body)
			if err != nil {
				t.Fatalf("unexpected error reading response body: %s", err)
			}

			var res response
			err = json.Unmarshal(body, &res)
			if err != nil {
				t.Fatalf("error unmarshalling response: %s", err)
			}

			if len(res.Users) != len(tc.users) {
				t.Errorf("expected to get %d users, got %d", len(tc.users), len(res.Users))
			}
		})
	}
}

func TestFindUserById(t *testing.T) {
	gin.SetMode(gin.ReleaseMode)

	now := time.Now()

	tt := []struct {
		name         string
		users        []model.User
		id           string
		expectedUser model.User
		expectedCode int
	}{
		{
			name: "test single user",
			users: []model.User{
				{
					ID:        1,
					Name:      "single",
					Email:     "single@dockyards.io",
					CreatedAt: now,
					UpdatedAt: now,
				},
			},
			id: "1",
			expectedUser: model.User{
				ID:        1,
				Name:      "single",
				Email:     "single@dockyards.io",
				CreatedAt: now,
				UpdatedAt: now,
			},
			expectedCode: http.StatusOK,
		},
		{
			name: "test multiple users",
			users: []model.User{
				{
					ID:        1,
					Name:      "multiple1",
					Email:     "multiple1@dockyards.io",
					CreatedAt: now,
					UpdatedAt: now,
				},
				{
					ID:        uint(2),
					Name:      "multiple2",
					Email:     "multiple2@dockyards.io",
					CreatedAt: now,
					UpdatedAt: now,
				},
				{
					ID:        3,
					Name:      "multiple3",
					Email:     "multiple3@dockyards.io",
					CreatedAt: now,
					UpdatedAt: now,
				},
			},
			id: "2",
			expectedUser: model.User{
				ID:        uint(2),
				Name:      "multiple2",
				Email:     "multiple2@dockyards.io",
				CreatedAt: now,
				UpdatedAt: now,
			},
			expectedCode: http.StatusOK,
		},
	}

	type response struct {
		User model.User `json:"user"`
	}

	for _, tc := range tt {
		t.Run(tc.name, func(t *testing.T) {
			db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
			if err != nil {
				t.Fatalf("unexpected error creating db: %s", err)
			}
			db.AutoMigrate(&model.User{})

			for _, user := range tc.users {
				tx := db.Create(&user)
				if tx.Error != nil {
					t.Fatalf("unxepected error preparing test database: %s", err)
				}
			}

			h := handler{
				db: db,
			}

			r := gin.New()
			r.GET("/test/:id", h.FindUserById)

			w := httptest.NewRecorder()
			req, err := http.NewRequest(http.MethodGet, path.Join("/test", tc.id), nil)
			if err != nil {
				t.Fatalf("unexpected error preparing request: %s", err)
			}

			r.ServeHTTP(w, req)

			if w.Code != tc.expectedCode {
				t.Fatalf("expected code %d, got %d", tc.expectedCode, w.Code)
			}

			body, err := io.ReadAll(w.Body)
			if err != nil {
				t.Fatalf("unexpected error reading response body: %s", err)
			}

			var res response
			err = json.Unmarshal(body, &res)
			if err != nil {
				t.Fatalf("error unmarshalling response: %s", err)
			}

			if res.User.ID != tc.expectedUser.ID {
				t.Errorf("expected user id %d, got %d", tc.expectedUser.ID, res.User.ID)
			}

			if res.User.Email != tc.expectedUser.Email {
				t.Errorf("expected user email %s, got %s", tc.expectedUser.Email, res.User.Email)
			}
		})
	}
}

func TestUpdateUser(t *testing.T) {
	gin.SetMode(gin.ReleaseMode)

	tt := []struct {
		name     string
		id       string
		users    []model.User
		update   model.User
		expected model.User
	}{
		{
			name: "test email update",
			id:   "1",
			users: []model.User{
				{
					ID:    1,
					Name:  "multiple1",
					Email: "multiple1@dockyards.io",
				},
				{
					ID:    2,
					Name:  "multiple2",
					Email: "multiple2@dockyards.io",
				},
				{
					ID:    3,
					Name:  "multiple3",
					Email: "multiple3@dockyards.io",
				},
			},
			update: model.User{
				Email: "new@dockyards.io",
			},
			expected: model.User{
				ID:    1,
				Name:  "email",
				Email: "new@dockyards.io",
			},
		},
		{
			name: "test id update is ignored",
			id:   "1",
			users: []model.User{
				{
					ID:    1,
					Name:  "multiple1",
					Email: "multiple1@dockyards.io",
				},
				{
					ID:    2,
					Name:  "multiple2",
					Email: "multiple2@dockyards.io",
				},
				{
					ID:    3,
					Name:  "multiple3",
					Email: "multiple3@dockyards.io",
				},
			},
			update: model.User{
				ID: 2,
			},
			expected: model.User{
				ID:    1,
				Name:  "multiple1",
				Email: "multiple1@dockyards.io",
			},
		},
	}

	type response struct {
		User model.User `json:"user"`
	}

	for _, tc := range tt {
		t.Run(tc.name, func(t *testing.T) {
			db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
			if err != nil {
				t.Fatalf("unexpected error creating db: %s", err)
			}
			db.AutoMigrate(&model.User{})

			for _, user := range tc.users {
				tx := db.Create(&user)
				if tx.Error != nil {
					t.Fatalf("unxepected error preparing test database: %s", err)
				}
			}

			h := handler{
				db: db,
			}

			r := gin.New()
			r.PUT("/test/:id", h.UpdateUser)

			b, err := json.Marshal(tc.update)
			if err != nil {
				t.Fatalf("unexpected error marshalling update: %s", err)
			}

			w := httptest.NewRecorder()
			req, err := http.NewRequest(http.MethodPut, path.Join("/test", tc.id), bytes.NewBuffer(b))
			if err != nil {
				t.Fatalf("unexpected error preparing request: %s", err)
			}
			req.Header.Add("content-type", "application/json")

			r.ServeHTTP(w, req)

			if w.Code != http.StatusOK {
				t.Errorf("expected code %d, got %d", http.StatusOK, w.Code)
			}

			body, err := io.ReadAll(w.Body)
			if err != nil {
				t.Fatalf("unexpected error reading response body: %s", err)
			}

			var res response
			err = json.Unmarshal(body, &res)
			if err != nil {
				t.Fatalf("error unmarshalling response: %s", err)
			}

			if res.User.ID != tc.expected.ID {
				t.Errorf("expected user id %d, got %d", tc.expected.ID, res.User.ID)
			}

			if res.User.Email != tc.expected.Email {
				t.Errorf("expected user email %s, got %s", tc.expected.Email, res.User.Email)
			}

			var actual model.User
			tx := db.First(&actual, tc.id)
			if tx.Error != nil {
				t.Fatalf("unexpected error fetching user from database: %s", tx.Error)
			}
		})
	}
}

func TestDeleteUser(t *testing.T) {
	gin.SetMode(gin.ReleaseMode)

	tt := []struct {
		name  string
		users []model.User
		id    string
	}{
		{
			name: "test multiple users",
			users: []model.User{
				{
					ID:    1,
					Name:  "multiple1",
					Email: "multiple1@dockyards.io",
				},
				{
					ID:    2,
					Name:  "multiple2",
					Email: "multiple2@dockyards.io",
				},
				{
					ID:    3,
					Name:  "multiple3",
					Email: "multiple3@dockyards.io",
				},
			},
			id: "3",
		},
		{
			name: "test last user",
			users: []model.User{
				{
					ID:    99,
					Name:  "last",
					Email: "lastdockyards.io",
				},
			},
			id: "99",
		},
	}

	for _, tc := range tt {
		t.Run(tc.name, func(t *testing.T) {
			db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
			if err != nil {
				t.Fatalf("unexpected error creating db: %s", err)
			}
			db.AutoMigrate(&model.User{})

			for _, user := range tc.users {
				tx := db.Create(&user)
				if tx.Error != nil {
					t.Fatalf("unxepected error preparing test database: %s", tx.Error)
				}
			}

			h := handler{
				db: db,
			}

			r := gin.New()
			r.DELETE("/test/:id", h.DeleteUser)

			w := httptest.NewRecorder()
			req, err := http.NewRequest(http.MethodDelete, path.Join("/test", tc.id), nil)
			if err != nil {
				t.Fatalf("unexpected error preparing request: %s", err)
			}

			r.ServeHTTP(w, req)

			if w.Code != http.StatusOK {
				t.Fatalf("expected code %d, got %d", http.StatusOK, w.Code)
			}

			var users []model.User
			tx := db.Find(&users)
			if tx.Error != nil {
				t.Fatalf("unexpected error fetching users from database: %s", tx.Error)
			}

			if len(users) == len(tc.users) {
				t.Errorf("same amount of users in database after delete: %d", len(users))
			}
		})
	}
}
