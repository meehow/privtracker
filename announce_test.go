package main

import (
	"net/http/httptest"
	"testing"
)

func BenchmarkAnnounce(b *testing.B) {
	server := httptest.NewServer(router())
	client := server.Client()
	for i := 0; i < b.N; i++ {
		_, err := client.Get(server.URL + "/test/announce?port=1234")
		if err != nil {
			b.Fatal(err)
		}
	}
}
