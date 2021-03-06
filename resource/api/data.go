// Copyright 2015 Canonical Ltd.
// Licensed under the AGPLv3, see LICENCE file for details.

package api

// TODO(ericsnow) Eliminate the dependence on apiserver if possible.

import (
	"strings"
	"time"

	"github.com/juju/errors"
	charmresource "gopkg.in/juju/charm.v6-unstable/resource"
	"gopkg.in/juju/names.v2"
	"gopkg.in/macaroon.v1"

	"github.com/juju/juju/apiserver/params"
	"github.com/juju/juju/charmstore"
)

// ListResourcesArgs are the arguments for the ListResources endpoint.
type ListResourcesArgs params.Entities

// NewListResourcesArgs returns the arguments for the ListResources endpoint.
func NewListResourcesArgs(services []string) (ListResourcesArgs, error) {
	var args ListResourcesArgs
	var errs []error
	for _, service := range services {
		if !names.IsValidApplication(service) {
			err := errors.Errorf("invalid application %q", service)
			errs = append(errs, err)
			continue
		}
		args.Entities = append(args.Entities, params.Entity{
			Tag: names.NewApplicationTag(service).String(),
		})
	}
	if err := resolveErrors(errs); err != nil {
		return args, errors.Trace(err)
	}
	return args, nil
}

// AddPendingResourcesArgs holds the arguments to the AddPendingResources
// API endpoint.
type AddPendingResourcesArgs struct {
	params.Entity
	params.AddCharmWithAuthorization

	// Resources is the list of resources to add as pending.
	Resources []CharmResource
}

// NewAddPendingResourcesArgs returns the arguments for the
// AddPendingResources API endpoint.
func NewAddPendingResourcesArgs(applicationID string, chID charmstore.CharmID, csMac *macaroon.Macaroon, resources []charmresource.Resource) (AddPendingResourcesArgs, error) {
	var args AddPendingResourcesArgs

	if !names.IsValidApplication(applicationID) {
		return args, errors.Errorf("invalid application %q", applicationID)
	}
	tag := names.NewApplicationTag(applicationID).String()

	var apiResources []CharmResource
	for _, res := range resources {
		if err := res.Validate(); err != nil {
			return args, errors.Trace(err)
		}
		apiRes := CharmResource2API(res)
		apiResources = append(apiResources, apiRes)
	}
	args.Tag = tag
	args.Resources = apiResources
	if chID.URL != nil {
		args.URL = chID.URL.String()
		args.Channel = string(chID.Channel)
		args.CharmStoreMacaroon = csMac
	}
	return args, nil
}

// AddPendingResourcesResult holds the result of the AddPendingResources
// API endpoint.
type AddPendingResourcesResult struct {
	params.ErrorResult

	// PendingIDs holds the "pending ID" for each of the requested
	// resources.
	PendingIDs []string `json:"pending-ids"`
}

// ResourcesResults holds the resources that result
// from a bulk API call.
type ResourcesResults struct {
	// Results is the list of resource results.
	Results []ResourcesResult `json:"results"`
}

// ResourcesResult holds the resources that result from an API call
// for a single application.
type ResourcesResult struct {
	params.ErrorResult

	// Resources is the list of resources for the application.
	Resources []Resource `json:"resources"`

	// CharmStoreResources is the list of resources associated with the charm in
	// the charmstore.
	CharmStoreResources []CharmResource `json:"charm-store-resources"`

	// UnitResources contains a list of the resources for each unit in the
	// application.
	UnitResources []UnitResources `json:"unit-resources"`
}

// A UnitResources contains a list of the resources the unit defined by Entity.
type UnitResources struct {
	params.Entity

	// Resources is a list of resources for the unit.
	Resources []Resource `json:"resources"`

	// DownloadProgress indicates the number of bytes of a resource file
	// have been downloaded so far the uniter. Only currently downloading
	// resources are included.
	DownloadProgress map[string]int64 `json:"download-progress"`
}

// UploadResult is the response from an upload request.
type UploadResult struct {
	params.ErrorResult

	// Resource describes the resource that was stored in the model.
	Resource Resource `json:"resource"`
}

// Resource contains info about a Resource.
type Resource struct {
	CharmResource

	// ID uniquely identifies a resource-application pair within the model.
	// Note that the model ignores pending resources (those with a
	// pending ID) except for in a few clearly pending-related places.
	ID string `json:"id"`

	// PendingID identifies that this resource is pending and
	// distinguishes it from other pending resources with the same model
	// ID (and from the active resource).
	PendingID string `json:"pending-id"`

	// ApplicationID identifies the application for the resource.
	ApplicationID string `json:"application"`

	// Username is the ID of the user that added the revision
	// to the model (whether implicitly or explicitly).
	Username string `json:"username"`

	// Timestamp indicates when the resource was added to the model.
	Timestamp time.Time `json:"timestamp"`
}

// CharmResource contains the definition for a resource.
type CharmResource struct {
	// Name identifies the resource.
	Name string `json:"name"`

	// Type is the name of the resource type.
	Type string `json:"type"`

	// Path is where the resource will be stored.
	Path string `json:"path"`

	// Description contains user-facing info about the resource.
	Description string `json:"description,omitempty"`

	// Origin is where the resource will come from.
	Origin string `json:"origin"`

	// Revision is the revision, if applicable.
	Revision int `json:"revision"`

	// Fingerprint is the SHA-384 checksum for the resource blob.
	Fingerprint []byte `json:"fingerprint"`

	// Size is the size of the resource, in bytes.
	Size int64 `json:"size"`
}

func resolveErrors(errs []error) error {
	switch len(errs) {
	case 0:
		return nil
	case 1:
		return errs[0]
	default:
		msgs := make([]string, len(errs))
		for i, err := range errs {
			msgs[i] = err.Error()
		}
		return errors.New(strings.Join(msgs, "\n"))
	}
}
