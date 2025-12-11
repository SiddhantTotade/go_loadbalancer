package strategy

import "../backend"

type Strategy interface {
	Next([]*backend.Backend) *backend.Backend
}
