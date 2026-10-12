//go:build live

package client

import (
	"reflect"
	"testing"
)

// Read-only checks of the uploaded revenue document lists, never deletes

func TestLiveRevenueDocuments(t *testing.T) {
	c := liveClient(t)

	queued, err := c.ListQueuedRevenues(0, 50, "")
	if err != nil {
		t.Fatalf("ListQueuedRevenues: %v", err)
	}
	if len(queued.Items) > 0 {
		raw := fetchRawList(t, c, "/proxy/accounting/revenues/queued", "/revenues")
		assertSchema(t, raw[0], reflect.TypeOf(QueuedRevenue{}), "queued revenue")
	}

	rejected, err := c.ListRejectedRevenues(0, 50)
	if err != nil {
		t.Fatalf("ListRejectedRevenues: %v", err)
	}
	if len(rejected.Items) > 0 {
		raw := fetchRawList(t, c, "/proxy/accounting/revenues/rejected", "/revenues")
		assertSchema(t, raw[0], reflect.TypeOf(RejectedRevenue{}), "rejected revenue")
	}
	t.Logf("revenue documents: %d queued, %d rejected", len(queued.Items), len(rejected.Items))
}
