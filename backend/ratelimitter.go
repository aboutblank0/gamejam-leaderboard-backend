package main

import (
	"net"
	"net/http"
	"sync"
	"time"
)

type Client struct {
	Address     string
	LastSeen    time.Time
	BannedUntil time.Time

	RequestCount int
	WindowStart  time.Time

	LastPost time.Time
}

var (
	clients = make(map[string]*Client)
	mu      sync.Mutex
)

const (
	getLimit     = 50
	getWindow    = 30 * time.Second
	postCooldown = 30 * time.Second
	banDuration  = 1 * time.Minute
)

const postCooldownMS time.Duration = 30 * time.Second

func getIp(r *http.Request) string {
	ip, _, err := net.SplitHostPort(r.RemoteAddr)
	if err != nil {
		return r.RemoteAddr
	}
	return ip
}

// Rate limitting Get Requests middleware
// Only allow 50 requests every 30 seconds
func RateLimitGet(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Check the IPs,
		ip := getIp(r)

		mu.Lock()

		now := time.Now()

		// Mark when last seen
		client, ok := clients[ip]
		if !ok {
			client = &Client{
				Address:     ip,
				LastSeen:    now,
				WindowStart: now,
			}
			clients[ip] = client
		}

		client.LastSeen = now

		var limitMsg string

		// Limit Banned IPs
		if now.Before(client.BannedUntil) {
			limitMsg = "Rate limit exceeded"
		} else {
			// Reset Window
			if now.Sub(client.WindowStart) > getWindow {
				client.WindowStart = now
				client.RequestCount = 0
			}

			client.RequestCount++

			if client.RequestCount > getLimit {
				client.BannedUntil = now.Add(banDuration)
				limitMsg = "Too many requests"
			}
		}

		mu.Unlock()

		if limitMsg != "" {
			http.Error(w, limitMsg, http.StatusTooManyRequests)
			return
		}

		next.ServeHTTP(w, r)
	})
}

// Rate limitting Post Requests middleware
// Only allow one request per IP every 30 seconds
func RateLimitPost(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ip := getIp(r)

		mu.Lock()

		now := time.Now()

		client, ok := clients[ip]
		if !ok {
			client = &Client{
				Address:     ip,
			}
			clients[ip] = client
		}

		client.LastSeen = now

		// Check the cooldown
		remaining := postCooldown - now.Sub(client.LastPost)
		if remaining > 0 {
			mu.Unlock()

			http.Error(
				w,
				"Please wait "+remaining.Round(time.Second).String(),
				http.StatusTooManyRequests,
			)
			return
		}

		client.LastPost = now

		mu.Unlock()
		next.ServeHTTP(w, r)
	})
}
