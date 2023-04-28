package handlers

import (
	"encoding/json"
	"errors"
	"net/http"
	"os"
	"path"
	"time"

	"bitbucket.org/sudosweden/backend/api/v1/model"
	"bitbucket.org/sudosweden/backend/internal"
	"github.com/docker/distribution/reference"
	"github.com/gin-gonic/gin"
	"github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/plumbing/object"
	"gorm.io/gorm"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/intstr"
)

func (h *handler) createDeployment(app *model.App) (*appsv1.Deployment, error) {
	containerPort := 80
	if app.Port != 0 {
		containerPort = app.Port
	}

	containerPorts := []corev1.ContainerPort{
		{
			Name:          "http",
			Protocol:      corev1.ProtocolTCP,
			ContainerPort: int32(containerPort),
		},
	}

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
							Ports:           containerPorts,
						},
					},
				},
			},
		},
	}

	return &deployment, nil
}

func (h *handler) createService(app *model.App) (*corev1.Service, error) {
	service := corev1.Service{
		TypeMeta: metav1.TypeMeta{
			APIVersion: "v1",
			Kind:       "Service",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name: app.Name,
		},
		Spec: corev1.ServiceSpec{
			Ports: []corev1.ServicePort{
				{
					Name:       "http",
					Port:       80,
					TargetPort: intstr.FromString("http"),
					Protocol:   corev1.ProtocolTCP,
				},
			},
			Selector: map[string]string{
				"app.kubernetes.io/name": app.Name,
			},
		},
	}

	return &service, nil
}

func (h *handler) parseContainerImage(ref string) (string, error) {
	named, err := reference.ParseDockerRef(ref)
	if err != nil {
		return "", err
	}

	return named.Name(), nil
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

	normalizedName, err := h.parseContainerImage(app.ContainerImage)
	if err != nil {
		c.JSON(http.StatusUnprocessableEntity, gin.H{
			"error":   "container image is not valid",
			"name":    app.ContainerImage,
			"details": err.Error(),
		})
	}

	app.ContainerImage = normalizedName
	if app.Name == "" {
		base := path.Base(app.ContainerImage)
		app.Name = base
	}

	details, validName := internal.IsValidName(app.Name)
	if !validName {
		c.JSON(http.StatusUnprocessableEntity, gin.H{
			"error":   "name is not valid",
			"name":    app.Name,
			"details": details,
		})
		return
	}

	var existingApp model.App
	err = h.db.Take(&existingApp, "name = ? AND organization = ? AND cluster = ?", app.Name, app.Organization, app.Cluster).Error
	if err != nil {
		if !errors.Is(err, gorm.ErrRecordNotFound) {
			h.logger.Error("error taking app from database", "name", app.Name, "err", err)
			c.AbortWithStatus(http.StatusInternalServerError)
		}
	}

	if existingApp.Name == app.Name && existingApp.Organization == app.Organization && existingApp.Cluster == app.Cluster {
		h.logger.Debug("app already exists", "name", app.Name, "organization", app.Organization, "cluster", app.Cluster)
		c.AbortWithStatus(http.StatusConflict)
		return
	}

	deployment, err := h.createDeployment(&app)
	if err != nil {
		h.logger.Error("failed to create deployment", "name", app.Name, "err", err)
	}

	deploymentJson, err := json.Marshal(deployment)
	if err != nil {
		h.logger.Error("error marshalling deployment as json", "err", err)
		c.AbortWithStatus(http.StatusInternalServerError)
		return
	}

	service, err := h.createService(&app)
	if err != nil {
		h.logger.Error("error creating service", "name", app.Name, "err", err)
		c.AbortWithStatus(http.StatusInternalServerError)
		return
	}

	serviceJson, err := json.Marshal(service)
	if err != nil {
		h.logger.Error("error mashalling service as json", "err", err)
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

	_, err = file.Write(deploymentJson)
	if err != nil {
		h.logger.Error("error writing to file", "err", err)
		c.AbortWithStatus(http.StatusInternalServerError)
		return
	}

	file.Close()
	worktree.Add("deployment.json")

	filename = path.Join(repoPath, "service.json")
	file, err = os.Create(filename)
	if err != nil {
		h.logger.Error("error creating service file", "filename", filename, "err", err)
		c.AbortWithStatus(http.StatusInternalServerError)
		return
	}

	_, err = file.Write(serviceJson)
	if err != nil {
		h.logger.Error("error writing service json to file", "err", err)
		c.AbortWithStatus(http.StatusInternalServerError)
		return
	}

	file.Close()
	worktree.Add("service.json")

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

	err = h.db.Create(&app).Error
	if err != nil {
		h.logger.Error("error creating app in database", "name", app.Name, "err", err)
		c.AbortWithStatus(http.StatusInternalServerError)
		return
	}

	c.JSON(http.StatusCreated, app)
}

func (s *sudo) GetApps(c *gin.Context) {
	var apps []model.App
	err := s.db.Find(&apps).Error
	if err != nil {
		c.AbortWithStatus(http.StatusInternalServerError)
		return
	}

	c.JSON(http.StatusOK, apps)
}

func (h *handler) GetApps(c *gin.Context) {
	user, err := h.getUserFromContext(c)
	if err != nil {
		h.logger.Debug("error fetching user from context", "err", err)
		c.AbortWithStatus(http.StatusInternalServerError)
		return
	}

	var organizations []model.Organization
	err = h.db.Model(&user).Association("Organizations").Find(&organizations)
	if err != nil {
		c.AbortWithStatus(http.StatusInternalServerError)
		return
	}

	orgs := make(map[string]bool)
	for _, organization := range organizations {
		orgs[organization.Name] = true
	}

	var apps []model.App
	err = h.db.Find(&apps).Error
	if err != nil {
		h.logger.Error("error finding apps in database", "err", err)
		c.AbortWithStatus(http.StatusInternalServerError)
		return
	}

	filteredApps := []model.App{}
	for _, app := range apps {
		_, isMember := orgs[app.Organization]

		h.logger.Debug("checking app membership", "name", app.Name, "organization", app.Organization, "member", isMember)

		if !isMember {
			continue
		}

		filteredApps = append(filteredApps, app)
	}

	c.JSON(http.StatusOK, filteredApps)
}

func (h *handler) DeleteOrgApps(c *gin.Context) {
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

	appName := c.Param("app")
	if appName == "" {
		h.logger.Debug("app empty")
		c.AbortWithStatus(http.StatusBadRequest)
		return
	}

	var app model.App
	err := h.db.Take(&app, "name = ? AND organization = ? AND cluster = ?", appName, org, cluster).Error
	if err != nil {
		h.logger.Error("error taking app from database", "err", err)
		c.AbortWithStatus(http.StatusInternalServerError)
		return
	}

	h.logger.Debug("deleting app from database", "id", app.ID)

	err = h.db.Delete(&app).Error
	if err != nil {
		h.logger.Error("error deleting app from database", "id", app.ID, "err", err)
		c.AbortWithStatus(http.StatusInternalServerError)
		return
	}

	h.logger.Debug("deleted app from database", "id", app.ID)

	repoPath := path.Join("/tmp/repos/v1", "orgs", org, "clusters", cluster, "apps", appName)

	h.logger.Debug("deleting repository from filesystem", "path", repoPath)

	err = os.RemoveAll(repoPath)
	if err != nil {
		h.logger.Error("error deleting repository from filesystem", "path", repoPath, "err", err)
		return
	}

	h.logger.Debug("deleted repository from filesystem", "path", repoPath)

}
