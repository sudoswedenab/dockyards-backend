package handlers

import (
	"encoding/json"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"os"
	"reflect"
	"testing"

	"bitbucket.org/sudosweden/dockyards-backend/api/v1"
	"bitbucket.org/sudosweden/dockyards-backend/internal/loggers"
	"bitbucket.org/sudosweden/dockyards-backend/internal/util"
	"github.com/gin-gonic/gin"
	"github.com/glebarez/sqlite"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

func TestGetApps(t *testing.T) {
	tt := []struct {
		name string
		apps []v1.App
	}{
		{
			name: "test empty",
			apps: []v1.App{},
		},
		{
			name: "test single",
			apps: []v1.App{
				{
					Id:   uuid.MustParse("86ea7a7c-2c77-49a8-9af2-a36be89aa031"),
					Name: "test",
				},
			},
		},
		{
			name: "test multiple",
			apps: []v1.App{
				{
					Id:   uuid.MustParse("7a8991b6-0fc8-450b-b97b-d39becc24d89"),
					Name: "test1",
				},
				{
					Id:   uuid.MustParse("3f09378e-c762-4725-9c28-443055297e75"),
					Name: "test2",
				},
				{
					Id:   uuid.MustParse("3f72e332-2148-44f3-9266-9f4793c5cf7f"),
					Name: "test3",
				},
			},
		},
	}

	gin.SetMode(gin.TestMode)

	for _, tc := range tt {
		t.Run(tc.name, func(t *testing.T) {
			logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelError + 1}))
			gormSlogger := loggers.NewGormSlogger(logger)

			db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{Logger: gormSlogger})
			if err != nil {
				t.Fatalf("unexpected error creating test database: %s", err)
			}
			db.AutoMigrate(&v1.App{})

			for _, app := range tc.apps {
				err := db.Create(&app).Error
				if err != nil {
					t.Fatalf("unexpected error creating app in test database: %s", err)
				}
			}

			h := handler{
				logger: logger,
				db:     db,
			}

			w := httptest.NewRecorder()
			c, _ := gin.CreateTestContext(w)

			h.GetApps(c)

			statusCode := w.Result().StatusCode
			if statusCode != http.StatusOK {
				t.Fatalf("expected status code %d, got %d", http.StatusOK, statusCode)
			}

			body, err := io.ReadAll(w.Result().Body)
			if err != nil {
				t.Fatalf("error reading result body: %s", err)
			}

			var actual []v1.App
			err = json.Unmarshal(body, &actual)
			if err != nil {
				t.Fatalf("error unmarshalling result body json: %s", err)
			}

			if !reflect.DeepEqual(actual, tc.apps) {
				t.Errorf("expected %#v, got %#v", tc.apps, actual)
			}
		})
	}
}

func TestGetAppsErrors(t *testing.T) {
}

func TestGetApp(t *testing.T) {
	tt := []struct {
		name        string
		appID       string
		apps        []v1.App
		appSteps    []v1.AppStep
		stepOptions []v1.StepOption
		expected    v1.App
	}{
		{
			name:  "test single",
			appID: "7d5fcf7d-e7aa-43da-83e7-700ffc37748e",
			apps: []v1.App{
				{
					Id:   uuid.MustParse("7d5fcf7d-e7aa-43da-83e7-700ffc37748e"),
					Name: "test",
				},
			},
			appSteps: []v1.AppStep{
				{
					Id:    uuid.MustParse("caf03733-fb00-4852-bb7f-849eb1e46539"),
					AppId: uuid.MustParse("7d5fcf7d-e7aa-43da-83e7-700ffc37748e"),
					Name:  "step",
				},
			},
			stepOptions: []v1.StepOption{
				{
					AppStepId:   uuid.MustParse("caf03733-fb00-4852-bb7f-849eb1e46539"),
					DisplayName: "Helm Chart",
					JsonPointer: "/helm_chart",
					Default:     util.Ptr("test"),
				},
				{
					AppStepId:   uuid.MustParse("caf03733-fb00-4852-bb7f-849eb1e46539"),
					DisplayName: "Helm Repository",
					JsonPointer: "/helm_repository",
					Default:     util.Ptr("http://localhost/chart-repository"),
				},
				{
					AppStepId:   uuid.MustParse("caf03733-fb00-4852-bb7f-849eb1e46539"),
					DisplayName: "Helm Version",
					JsonPointer: "/helm_version",
					Default:     util.Ptr("1.2.3"),
				},
			},
			expected: v1.App{
				Id:   uuid.MustParse("7d5fcf7d-e7aa-43da-83e7-700ffc37748e"),
				Name: "test",
				AppSteps: []v1.AppStep{
					{
						Id:    uuid.MustParse("caf03733-fb00-4852-bb7f-849eb1e46539"),
						AppId: uuid.MustParse("7d5fcf7d-e7aa-43da-83e7-700ffc37748e"),
						Name:  "step",
						StepOptions: []v1.StepOption{
							{
								AppStepId:   uuid.MustParse("caf03733-fb00-4852-bb7f-849eb1e46539"),
								DisplayName: "Helm Chart",
								JsonPointer: "/helm_chart",
								Default:     util.Ptr("test"),
							},
							{
								AppStepId:   uuid.MustParse("caf03733-fb00-4852-bb7f-849eb1e46539"),
								DisplayName: "Helm Repository",
								JsonPointer: "/helm_repository",
								Default:     util.Ptr("http://localhost/chart-repository"),
							},
							{
								AppStepId:   uuid.MustParse("caf03733-fb00-4852-bb7f-849eb1e46539"),
								DisplayName: "Helm Version",
								JsonPointer: "/helm_version",
								Default:     util.Ptr("1.2.3"),
							},
						},
					},
				},
			},
		},
		{
			name:  "test unused app options",
			appID: "d837640f-01e4-4834-a82c-2c7c995afdb0",
			apps: []v1.App{
				{
					Id:   uuid.MustParse("d837640f-01e4-4834-a82c-2c7c995afdb0"),
					Name: "test1",
				},
				{
					Id:   uuid.MustParse("657bb8da-4018-4457-a295-325a295c803b"),
					Name: "test2",
				},
			},
			appSteps: []v1.AppStep{
				{
					Id:    uuid.MustParse("fe0e09c4-e4c8-4e76-99a2-8b6d9f07df02"),
					AppId: uuid.MustParse("d837640f-01e4-4834-a82c-2c7c995afdb0"),
				},
				{
					Id:    uuid.MustParse("2ee1a20a-662b-46ed-bac1-f1824466aa6f"),
					AppId: uuid.MustParse("657bb8da-4018-4457-a295-325a295c803b"),
				},
			},
			stepOptions: []v1.StepOption{
				{
					AppStepId:   uuid.MustParse("fe0e09c4-e4c8-4e76-99a2-8b6d9f07df02"),
					JsonPointer: "/helm_chart",
					Default:     util.Ptr("test1"),
				},
				{
					AppStepId:   uuid.MustParse("2ee1a20a-662b-46ed-bac1-f1824466aa6f"),
					JsonPointer: "/helm_chart",
					Default:     util.Ptr("test2"),
				},
				{
					AppStepId:   uuid.MustParse("2ee1a20a-662b-46ed-bac1-f1824466aa6f"),
					JsonPointer: "/helm_repository",
					Default:     util.Ptr("test2"),
				},
			},
			expected: v1.App{
				Id:   uuid.MustParse("d837640f-01e4-4834-a82c-2c7c995afdb0"),
				Name: "test1",
				AppSteps: []v1.AppStep{
					{
						Id:    uuid.MustParse("fe0e09c4-e4c8-4e76-99a2-8b6d9f07df02"),
						AppId: uuid.MustParse("d837640f-01e4-4834-a82c-2c7c995afdb0"),
						StepOptions: []v1.StepOption{
							{
								AppStepId:   uuid.MustParse("fe0e09c4-e4c8-4e76-99a2-8b6d9f07df02"),
								JsonPointer: "/helm_chart",
								Default:     util.Ptr("test1"),
							},
						},
					},
				},
			},
		},
	}

	for _, tc := range tt {
		t.Run(tc.name, func(t *testing.T) {
			logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelError + 1}))
			gormSlogger := loggers.NewGormSlogger(logger)

			db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{Logger: gormSlogger})
			if err != nil {
				t.Fatalf("unexpected error creating test database: %s", err)
			}
			db.AutoMigrate(&v1.App{})
			db.AutoMigrate(&v1.AppStep{})
			db.AutoMigrate(&v1.StepOption{})
			for _, app := range tc.apps {
				err := db.Create(&app).Error
				if err != nil {
					t.Fatalf("unexpected error creating app in test database: %s", err)
				}
			}
			for _, appStep := range tc.appSteps {
				err := db.Create(&appStep).Error
				if err != nil {
					t.Fatalf("unxepected error creating app step in test database: %s", err)
				}
			}
			for _, stepOption := range tc.stepOptions {
				err := db.Create(&stepOption).Error
				if err != nil {
					t.Fatalf("unxepected error creating step option in test database: %s", err)
				}
			}

			h := handler{
				logger: logger,
				db:     db,
			}

			w := httptest.NewRecorder()
			c, _ := gin.CreateTestContext(w)
			c.Params = []gin.Param{
				{Key: "appID", Value: tc.appID},
			}

			h.GetApp(c)

			statusCode := w.Result().StatusCode
			if statusCode != http.StatusOK {
				t.Fatalf("expected status code %d, got %d", http.StatusOK, statusCode)
			}

			body, err := io.ReadAll(w.Result().Body)
			if err != nil {
				t.Fatalf("error reading result body: %s", err)
			}

			var actual v1.App
			err = json.Unmarshal(body, &actual)
			if err != nil {
				t.Fatalf("error unmarshalling result body json: %s", err)
			}

			if !reflect.DeepEqual(actual, tc.expected) {
				t.Errorf("expected\n %#v, got\n %#v", tc.expected, actual)
			}
		})
	}
}

func TestGetAppErrors(t *testing.T) {
}
