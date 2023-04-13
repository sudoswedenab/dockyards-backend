package rancher

import "github.com/rancher/norman/types"

func (r *Rancher) addGarbage(resource *types.Resource) {
	r.garbageMutex.Lock()
	r.garbageObjects[resource.ID] = resource
	r.garbageMutex.Unlock()
}

func (r *Rancher) DeleteGarbage() {
	r.Logger.Debug("delete garbage start", "objects", len(r.garbageObjects))

	r.garbageMutex.Lock()
	defer r.garbageMutex.Unlock()

	// deletion of each garbage object is only attemped once when called
	// if deletion fails, try again when called the next time
	for name, resource := range r.garbageObjects {
		r.Logger.Debug("delete garbage object", "name", name, "type", resource.Type)

		err := r.ManagementClient.APIBaseClient.Delete(resource)
		if err != nil {
			r.Logger.Debug("deleting garbage object failed", "name", name, "type", resource.Type, "err", err)
			continue
		}

		delete(r.garbageObjects, name)

		r.Logger.Debug("deleted garbage object", "name", name, "type", resource.Type)
	}

	r.Logger.Debug("delete garbage end", "objects", len(r.garbageObjects))
}
