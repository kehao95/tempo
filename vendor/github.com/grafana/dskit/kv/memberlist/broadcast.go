package memberlist

import (
	"fmt"

	"github.com/go-kit/kit/log"
	"github.com/go-kit/kit/log/level"
	"github.com/hashicorp/memberlist"
)

// ringBroadcast implements memberlist.Broadcast interface, which is used by memberlist.TransmitLimitedQueue.
type ringBroadcast struct {
	key      string
	content  []string // Description of what is stored in this value. Used for invalidation.
	version  uint     // local version of the value, generated by merging this change
	msg      []byte   // encoded key and value
	finished func(b ringBroadcast)
	logger   log.Logger
}

func (r ringBroadcast) Invalidates(old memberlist.Broadcast) bool {
	if oldb, ok := old.(ringBroadcast); ok {
		if r.key != oldb.key {
			return false
		}

		// if 'content' (result of Mergeable.MergeContent) of this broadcast is a superset of content of old value,
		// and this broadcast has resulted in a newer ring update, we can invalidate the old value

		for _, oldName := range oldb.content {
			found := false
			for _, newName := range r.content {
				if oldName == newName {
					found = true
					break
				}
			}

			if !found {
				return false
			}
		}

		// only do this check if this ringBroadcast covers same ingesters as 'b'
		// otherwise, we may be invalidating some older messages, which however covered different
		// ingesters
		if r.version >= oldb.version {
			level.Debug(r.logger).Log("msg", "Invalidating forwarded broadcast", "key", r.key, "version", r.version, "oldVersion", oldb.version, "content", fmt.Sprintf("%v", r.content), "oldContent", fmt.Sprintf("%v", oldb.content))
			return true
		}
	}
	return false
}

func (r ringBroadcast) Message() []byte {
	return r.msg
}

func (r ringBroadcast) Finished() {
	if r.finished != nil {
		r.finished(r)
	}
}
