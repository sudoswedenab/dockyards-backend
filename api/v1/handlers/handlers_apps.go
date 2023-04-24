package handlers

import (
	"encoding/json"
	"net/http"
	"os"
	"path"
	"time"

	"bitbucket.org/sudosweden/backend/api/v1/model"
	"bitbucket.org/sudosweden/backend/internal"
	"github.com/gin-gonic/gin"
	"github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/plumbing/object"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func (h *handler) createDeployment(app *model.App) (*appsv1.Deployment, error) {
	deployment := appsv1.Deployment{
		TypeMeta: metav1.TypeMeta{
			APIVersion: "apps/v1",
			Kind:       "Deployment",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name: app.Name,
		},
		Spec: appsv1.DeploymentSpec{
			Selector: &metav1.LabelSelector{
				MatchLabels: map[string]string{
					"app.kubernetes.io/name": app.Name,
				},
			},
			Template: corev1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Labels: map[string]string{
						"app.kubernetes.io/name": app.Name,
					},
				},
				Spec: corev1.PodSpec{
					Containers: []corev1.Container{
						{
							Name:            app.Name,
							Image:           app.ContainerImage,
							ImagePullPolicy: corev1.PullIfNotPresent,
							Ports: []corev1.ContainerPort{
								{
									Name:          "http",
									Protocol:      corev1.ProtocolTCP,
									ContainerPort: 80,
								},
							},
						},
					},
				},
			},
		},
	}

	return &deployment, nil
}

func (h *handler) PostOrgApps(c *gin.Context) {
	org := c.Param("org")
	if org == "" {
		h.logger.Debug("org empty")
		c.AbortWithStatus(http.StatusBadRequest)
		return
	}

	cluster := c.Param("cluster")
	if cluster == "" {
		h.logger.Debug("cluster empty")
		c.AbortWithStatus(http.StatusBadRequest)
		return
	}

	var app model.App
	err := c.BindJSON(&app)
	if err != nil {
		h.logger.Error("failed to read body", "err", err)
		c.AbortWithStatus(http.StatusBadRequest)
		return
	}

	app.Organization = org
	app.Cluster = cluster

	details, validName := internal.IsValidName(app.Name)
	if !validName {
		c.JSON(http.StatusBadRequest, gin.H{
			"error":   "name is not valid",
			"name":    app.Name,
			"details": details,
		})
		return
	}

	deployment, err := h.createDeployment(&app)
	if err != nil {
		h.logger.Error("failed to create deployment", "name", app.Name, "err", err)
	}

	b, err := json.Marshal(deployment)
	if err != nil {
		h.logger.Error("error marshalling deployment as json", "err", err)
		c.AbortWithStatus(http.StatusInternalServerError)
		return
	}

	repoPath := path.Join("/tmp/repos/v1", "orgs", org, "clusters", cluster, "apps", app.Name)

	repo, err := git.PlainInit(repoPath, false)
	if err != nil {
		h.logger.Error("error creating git repository", "path", repoPath, "err", err)
		c.AbortWithStatus(http.StatusInternalServerError)
		return
	}
	worktree, err := repo.Worktree()
	if err != nil {
		h.logger.Error("error getting default worktree", "err", err)
		c.AbortWithStatus(http.StatusInternalServerError)
		return
	}

	filename := path.Join(repoPath, "deployment.json")
	file, err := os.Create(filename)
	if err != nil {
		h.logger.Error("error creating deployment file", "filename", filename, "err", err)
		c.AbortWithStatus(http.StatusInternalServerError)
		return
	}
	defer file.Close()

	_, err = file.Write(b)
	if err != nil {
		h.logger.Error("error writing to file", "err", err)
		c.AbortWithStatus(http.StatusInternalServerError)
		return
	}

	worktree.Add("deployment.json")
	commit, err := worktree.Commit("Add deployment", &git.CommitOptions{
		Author: &object.Signature{
			Name:  "dockyards",
			Email: "git@dockyards.io",
			When:  time.Now(),
		},
	})

	if err != nil {
		h.logger.Error("error creating commit", "err", err)
		c.AbortWithStatus(http.StatusInternalServerError)
		return
	}

	h.logger.Debug("created commit", "hash", commit.String())

	h.db.Create(&app)
}
