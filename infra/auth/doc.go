// Package auth authenticates and authorizes gRPC calls of Webitel services.
//
// The access token (metadata key x-webitel-access) is exchanged for a Session
// through Auth.UserInfo of go.webitel.app. The interceptor checks the call
// against the generated WebitelAPI map: licenses first, then obac for the
// object class and action; methods missing from the map are denied. Handlers
// read the caller with SessionFromContext.
//
// See the package example for wiring.
package auth
