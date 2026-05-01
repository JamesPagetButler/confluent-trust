// Package store contains storage backends for CTH inventories.
//
//   - json.go: file-backed JSON store (Crawl phase, default)
//   - muninn.go (Walk): MuninnDB engram store with Hebbian co-activation
//   - surreal.go (Run): SurrealDB structural ground truth + vector search
//
// During the Crawl phase only the JSON backend ships; the package does not
// expose a Store interface yet — that lands when a second backend appears.
package store
