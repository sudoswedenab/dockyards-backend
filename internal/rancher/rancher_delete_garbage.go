package rancher

import "github.com/rancher/norman/types"

func (r *rancher) addGarbage(resource *types.Resource) {
	r.garbageMutex.Lock()
	r.garbageObjects[resource.ID] = resource
	r.garbageMutex.Unlock()
}

func (r *rancher) DeleteGarbage() {
	r.logger.Debug("delete garbage start", "objects", len(r.garbageObjects))

	r.garbageMutex.Lock()
	defer r.garbageMutex.Unlock()

	// deletion of each garbage object is only attemped once when called
	// if deletion fails, try again when called the next time
	for name, resource := range r.garbageObjects {
		r.logger.Debug("delete garbage object", "name", name, "type", resource.Type)

		err := r.managementClient.APIBaseClient.Delete(resource)
		if err != nil {
			r.logger.Debug("deleting garbage object failed", "name", name, "type", resource.Type, "err", err)
			continue
		}

		delete(r.garbageObjects, name)

		r.logger.Debug("deleted garbage object", "name", name, "type", resource.Type)
	}

	r.logger.Debug("delete garbage end", "objects", len(r.garbageObjects))
}
