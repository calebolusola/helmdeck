package api

// Adapter that wires session.Runtime into mcp.SessionLister so the MCP
// server can serve helmdeck://sessions as a resource (issue #44 / ADR 006).
//
// Kept narrow on purpose: PackServer doesn't need (or want) the full
// session.Runtime API (Create / Logs / Delete) — only List. Limiting
// the surface keeps MCP from becoming a back-channel for session
// mutation that would bypass the audit log on /api/v1/sessions/*.

import (
	"context"
	"time"

	"github.com/tosin2013/helmdeck/internal/mcp"
	"github.com/tosin2013/helmdeck/internal/session"
)

type sessionListerAdapter struct {
	rt session.Runtime
}

func (a sessionListerAdapter) List(ctx context.Context) ([]mcp.SessionView, error) {
	sessions, err := a.rt.List(ctx)
	if err != nil {
		return nil, err
	}
	views := make([]mcp.SessionView, 0, len(sessions))
	for _, s := range sessions {
		views = append(views, mcp.SessionView{
			ID:        s.ID,
			Status:    string(s.Status),
			Image:     s.Spec.Image,
			CreatedAt: s.CreatedAt.UTC().Format(time.RFC3339),
		})
	}
	return views, nil
}
