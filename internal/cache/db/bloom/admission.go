package bloom

import (
	"github.com/Borislavv/go-ash-cache/config"
)

type AdmissionControl interface {
	Record(h uint64)
	Allow(candidate, victim uint64) bool
	Estimate(h uint64) uint8
	Reset()
}

func NewAdmissionControl(cfg *config.AdmissionControlCfg) AdmissionControl {
	if cfg.Enabled() {
		return newShardedAdmitter(cfg)
	} else {
		return newNoOp()
	}
}
