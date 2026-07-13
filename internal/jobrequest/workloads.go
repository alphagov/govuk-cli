package jobrequest

import (
	"fmt"
	"strings"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"
)

// Resolve a resource kind string to an APIResource
func (c *JobRequestClient) ResolveWorkload(resource string) (*metav1.APIResource, error) {
	resource = strings.ToLower(strings.TrimSpace(resource))
	if resource == "" {
		return nil, fmt.Errorf("no resource kind supplied")
	}

	var wantGroup string
	if name, group, found := strings.Cut(resource, "."); found {
		resource, wantGroup = name, group
	}

	lists, err := c.clientSet.Discovery().ServerPreferredResources()
	if err != nil && len(lists) == 0 {
		return nil, fmt.Errorf("discovering server resources: %w", err)
	}

	var matches []*metav1.APIResource
	for _, list := range lists {
		gv, parseErr := schema.ParseGroupVersion(list.GroupVersion)
		if parseErr != nil {
			continue
		}
		if wantGroup != "" && gv.Group != wantGroup {
			continue
		}

		for i := range list.APIResources {
			r := list.APIResources[i]
			// Subresources (e.g. "deployments/status") contain a slash; skip them.
			if strings.Contains(r.Name, "/") {
				continue
			}
			if !resourceMatches(r, resource) {
				continue
			}

			// APIResource entries inherit their group/version from the list.
			r.Group = gv.Group
			r.Version = gv.Version
			matches = append(matches, &r)
		}
	}

	switch len(matches) {
	case 0:
		return nil, fmt.Errorf("cluster doesn't have a resource with name %q", resource)
	case 1:
		return matches[0], nil
	default:
		return nil, ambiguousError(resource, matches)
	}
}

func resourceMatches(r metav1.APIResource, resource string) bool {
	if strings.ToLower(r.Name) == resource ||
		strings.ToLower(r.SingularName) == resource ||
		strings.ToLower(r.Kind) == resource {
		return true
	}
	for _, short := range r.ShortNames {
		if strings.ToLower(short) == resource {
			return true
		}
	}
	return false
}

func ambiguousError(resource string, matches []*metav1.APIResource) error {
	options := make([]string, 0, len(matches))
	for _, m := range matches {
		options = append(options, fmt.Sprintf("%s.%s", m.Name, m.Group))
	}
	return fmt.Errorf("%q is ambiguous, matched %s; qualify it as resource.group", resource, strings.Join(options, ", "))
}
