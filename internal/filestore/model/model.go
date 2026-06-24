package model

type FileStatus string

const (
	StatusPending FileStatus = "pending"
	StatusReady   FileStatus = "ready"
	StatusExpired FileStatus = "expired"
	StatusDeleted FileStatus = "deleted"
)

func (s FileStatus) String() string { return string(s) }

type Visibility string

const (
	VisPublic Visibility = "public"
	VisAgent  Visibility = "agent"
	VisToken  Visibility = "token"
)

func (v Visibility) String() string { return string(v) }

const (
	RouteUpload = "/upload/"
	RouteFile   = "/f/"
	RouteMCP    = "/mcp"
	RouteHealth = "/health"
)

const (
	ErrNotAuthenticated = "not authenticated"
	ErrForbidden        = "forbidden"
	ErrFileNotFound     = "file not found"
)
