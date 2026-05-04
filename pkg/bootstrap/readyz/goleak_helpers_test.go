package readyz

import "go.uber.org/goleak"

// goleakIgnores declares goroutines we intentionally exclude from leak
// detection. These are parked by third-party libraries (fasthttp) at process
// scope — once the first fiber.App.Test runs, a singleton goroutine loops
// forever updating Date headers. It is not owned by any /readyz code path.
//
// Lives in an untagged file so it is available to every *_test.go in this
// package regardless of build tags (chaos, e2e, default).
func goleakIgnores() []goleak.Option {
	return []goleak.Option{
		// fasthttp parks a permanent "update server date" goroutine once its
		// HTTP server code is first exercised. It is a well-known background
		// task documented by goleak users across the ecosystem. Using
		// IgnoreAnyFunction because the goroutine's top-of-stack is the stdlib
		// time.Sleep — the signature frame is updateServerDate.func1 deeper in.
		goleak.IgnoreAnyFunction("github.com/valyala/fasthttp.updateServerDate.func1"),
	}
}
