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
//
// # Generation and condition freshness
//
// The store assigns Generation 1 when it creates a desire. Generation advances
// only when Apply spec content changes, independently of the CAS Version, which
// advances on accepted Apply spec updates and on Apply and Delete status writes.
// The Apply controller stamps
// the generation it processed into the Successful condition's
// ObservedGeneration. Compare those two fields on the same returned ApplyDesire:
// a missing condition means no reported outcome; a mismatch means the condition
// describes an earlier spec; a match means Status and Reason describe an attempt
// at the current spec. A matching generation alone does not imply success or
// Kubernetes workload readiness.
//
// Delete and Read desires also receive Generation 1 from the store, but have no
// spec-update operation. Their controllers do not yet stamp ObservedGeneration;
// HYPERFLEET-1725 tracks their adoption:
// https://redhat.atlassian.net/browse/HYPERFLEET-1725.
//
// Desire Generation and its condition's ObservedGeneration are independent of
// the HyperFleet API resource's generation (carried by the adapter in the
// manifest's hyperfleet.io/generation annotation) and the target Kubernetes
// object's metadata.generation and status observed-generation fields. Each
// desire record has its own counter; do not compare counters across sibling
// Apply, Delete, or Read desires. Recreating a desire starts a new counter.
//
// For example, a newly created ApplyDesire may have Generation 1 while its
// manifest carries API generation 7 and the live Kubernetes object has
// metadata.generation 3. The Apply condition reports ObservedGeneration 1 after
// an attempt at that desire spec, not 7 or 3.
//
// Adapter annotation-based create/update/skip decisions are covered separately
// by HYPERFLEET-1441: https://redhat.atlassian.net/browse/HYPERFLEET-1441.
// HYPERFLEET-1516 provides desire-condition correlation; it does not implement
// adapter policy consuming that correlation. Adapter transport policy belongs
// to HYPERFLEET-1419: https://redhat.atlassian.net/browse/HYPERFLEET-1419.
//
// Behavioral semantics are aligned with the ARO-HCP kube-applier specification.
//
// Store interfaces (SpecStore, StatusStore) live here; backends live in
// store/memory and store/redis.
package desire
