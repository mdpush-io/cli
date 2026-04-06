package auth

import (
	"fmt"

	"github.com/mdpush-io/cli/internal/api"
)

// GitHubStartResult is what RequestGitHubDevice returns to the caller
// (a TUI typically) so it can render the user code and start polling.
type GitHubStartResult struct {
	DeviceCode      string
	UserCode        string
	VerificationURI string
	Interval        int // seconds between polls
	ExpiresIn       int
}

// RequestGitHubDevice initiates the GitHub Device Flow against the mdpush
// backend. The caller displays UserCode + VerificationURI to the user and
// then loops PollGitHubDevice until it returns a session.
func RequestGitHubDevice(client *api.Client) (*GitHubStartResult, error) {
	resp, err := client.RequestGitHubDeviceCode()
	if err != nil {
		return nil, fmt.Errorf("starting GitHub device flow: %w", err)
	}
	if resp.Interval <= 0 {
		resp.Interval = 5
	}
	return &GitHubStartResult{
		DeviceCode:      resp.DeviceCode,
		UserCode:        resp.UserCode,
		VerificationURI: resp.VerificationURI,
		Interval:        resp.Interval,
		ExpiresIn:       resp.ExpiresIn,
	}, nil
}

// GitHubPollResult is one tick of the device-flow polling loop.
// Exactly one of (Session, Pending, Err) is meaningful:
//   - Session != nil           → user authorized, login complete
//   - Pending == true          → keep polling
//   - NewInterval > 0          → server asked us to slow down
//   - Err != nil               → terminal error (expired, denied, etc.)
type GitHubPollResult struct {
	Session     *Session
	Pending     bool
	NewInterval int
	Err         error
}

// PollGitHubDevice runs a single poll against the backend. The caller
// (TUI) is responsible for waiting `Interval` seconds between calls.
func PollGitHubDevice(client *api.Client, deviceCode string) GitHubPollResult {
	resp, err := client.PollGitHubDeviceToken(api.DeviceTokenRequest{
		DeviceCode:  deviceCode,
		DeviceLabel: deviceLabel(),
	})
	if err != nil {
		return GitHubPollResult{Err: fmt.Errorf("polling GitHub: %w", err)}
	}

	switch resp.Status {
	case "authorized":
		return GitHubPollResult{
			Session: &Session{
				Token:     resp.SessionToken,
				UserID:    resp.UserID,
				Email:     resp.Email,
				ExpiresAt: resp.ExpiresAt,
			},
		}
	case "pending":
		return GitHubPollResult{Pending: true}
	case "slow_down":
		interval := resp.Interval
		if interval <= 0 {
			interval = 10
		}
		return GitHubPollResult{Pending: true, NewInterval: interval}
	case "expired":
		return GitHubPollResult{Err: fmt.Errorf("the GitHub login code expired — please try again")}
	case "denied":
		return GitHubPollResult{Err: fmt.Errorf("GitHub login was denied")}
	case "error":
		return GitHubPollResult{Err: fmt.Errorf("GitHub login error: %s", resp.Error)}
	default:
		return GitHubPollResult{Err: fmt.Errorf("unexpected status from server: %q", resp.Status)}
	}
}
