// Package dockerfile provides utilities for parsing multi-stage Dockerfiles
// used by agentbox's update workflow. It can split a Dockerfile at the boundary
// between the agentbox-managed stage and the user-managed custom stage.
package dockerfile
