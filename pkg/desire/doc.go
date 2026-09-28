// Package desire defines the core desire types for the desire-based delivery system.
//
// The desire model replaces Maestro/OCM ManifestWork as the transport mechanism
// for delivering Kubernetes resources to target clusters. Three desire types exist:
//
//   - ApplyDesire: make a resource exist with specific content (SSA force=true)
//   - DeleteDesire: make a resource not exist (confirmed gone past finalizers)
//   - ReadDesire: mirror a live object's state back to the control plane
//
// Each desire targets one Kubernetes resource (Identity). The store also
// supports listing and prefix delete (ListApplyDesires, DeleteByPrefix).
// Generation starts at 1 and advances on successful Apply spec updates,
// independently of the CAS Version, which also advances on Apply and Delete
// status writes. The Apply controller stamps the generation it processed into
// the Successful condition's ObservedGeneration. Consumers compare the two
// fields on the same desire to determine whether a condition is current.
// Delete and Read controllers do not yet stamp ObservedGeneration.
//
// Behavioral semantics are aligned with the ARO-HCP kube-applier specification.
//
// Store interfaces (SpecStore, StatusStore) live here; backends live in
// store/memory and store/redis.
package desire
