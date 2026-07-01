package solarman

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/chromedp/cdproto/network"
	"github.com/chromedp/chromedp"
)

// CaptureToken launches a real, visible Chrome window at the Solarman login,
// lets the user sign in manually (solving the Cloudflare/captcha challenge),
// and captures the Bearer JWT from the first authenticated API request that
// Solarman's page fires after login. Returns the token, or an error/timeout.
//
// This needs Google Chrome (or a Chromium/Edge) installed — chromedp locates it
// automatically. It's read-only: the user logs into their own account.
func CaptureToken(timeout time.Duration) (string, error) {
	opts := append(chromedp.DefaultExecAllocatorOptions[:],
		chromedp.Flag("headless", false),
		chromedp.Flag("start-maximized", true),
		// look less like automation so the captcha behaves normally
		chromedp.Flag("enable-automation", false),
		chromedp.Flag("disable-blink-features", "AutomationControlled"),
		chromedp.Flag("excludeSwitches", "enable-automation"),
	)
	allocCtx, cancelAlloc := chromedp.NewExecAllocator(context.Background(), opts...)
	defer cancelAlloc()
	ctx, cancel := chromedp.NewContext(allocCtx)
	defer cancel()

	tokenCh := make(chan string, 1)
	chromedp.ListenTarget(ctx, func(ev interface{}) {
		e, ok := ev.(*network.EventRequestWillBeSent)
		if !ok || !strings.Contains(e.Request.URL, "solarmanpv.com") {
			return
		}
		for k, v := range e.Request.Headers {
			if !strings.EqualFold(k, "authorization") {
				continue
			}
			if s, ok := v.(string); ok {
				s = strings.TrimSpace(strings.TrimPrefix(s, "Bearer "))
				if strings.HasPrefix(s, "eyJ") {
					select {
					case tokenCh <- s:
					default:
					}
				}
			}
		}
	})

	if err := chromedp.Run(ctx,
		network.Enable(),
		chromedp.Navigate("https://home.solarmanpv.com/login"),
	); err != nil {
		return "", fmt.Errorf("could not launch a browser (is Google Chrome installed?): %w", err)
	}

	select {
	case tok := <-tokenCh:
		return tok, nil
	case <-time.After(timeout):
		return "", fmt.Errorf("timed out waiting for login (%s)", timeout)
	case <-ctx.Done():
		return "", fmt.Errorf("login window closed before sign-in completed")
	}
}
