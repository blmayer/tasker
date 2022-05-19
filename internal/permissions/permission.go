package permissions

import "tasker/internal/types"

const (
	Zero = types.Permissions(1 << iota)

	// task related
	ReadTask
	WriteTask

	// list related
	PublicList

	// user related
	DeleteAccount
	CreateList
	DeleteList
)
