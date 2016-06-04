// Copyright 2016 Burak Sezer
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at

//     http://www.apache.org/licenses/LICENSE-2.0

// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package httptimeout

import (
	"crypto/tls"
	"net"
	"net/http"
	"time"
)

// Conn wraps a net.Conn, and sets a deadline for every read
// and write operation.
type Conn struct {
	net.Conn
	ReadTimeout  time.Duration
	WriteTimeout time.Duration
}

// Read wraps the net.Conn's original Read method.
func (c *Conn) Read(b []byte) (int, error) {
	err := c.Conn.SetReadDeadline(time.Now().Add(c.ReadTimeout))
	if err != nil {
		return 0, err
	}
	return c.Conn.Read(b)
}

// Write wraps the net.Conn's original Write method.
func (c *Conn) Write(b []byte) (int, error) {
	err := c.Conn.SetWriteDeadline(time.Now().Add(c.WriteTimeout))
	if err != nil {
		return 0, err
	}
	return c.Conn.Write(b)
}

// NewTransport returns http/Transport instance with timeout support
func NewTransport(addr string, readTimeout, writeTimeout time.Duration) *http.Transport {
	t := &http.Transport{}
	t.Dial = func(network, addr string) (net.Conn, error) {
		conn, err := net.DialTimeout(network, addr, readTimeout)
		if err != nil {
			return nil, err
		}
		tc := &Conn{
			Conn:         conn,
			ReadTimeout:  readTimeout,
			WriteTimeout: writeTimeout,
		}
		return tc, nil
	}
	return t
}

// Listener wraps a net.Listener, and gives a place to store the timeout
// parameters. On Accept, it will wrap the net.Conn with our own Conn for us.
type Listener struct {
	net.Listener
	ReadTimeout  time.Duration
	WriteTimeout time.Duration
}

// Accept wraps the Accept method of the original Listener. It waits for the next call and returns
// a Conn which wraps the net.Conn with timeout.
func (l *Listener) Accept() (net.Conn, error) {
	c, err := l.Listener.Accept()
	if err != nil {
		return nil, err
	}
	tc := &Conn{
		Conn:         c,
		ReadTimeout:  l.ReadTimeout,
		WriteTimeout: l.WriteTimeout,
	}
	return tc, nil
}

// NewListener runs net.Listen and announces on the network address addr with timeout.
// The network net must be a stream-oriented network: "tcp", "tcp4", "tcp6", "unix" or "unixpacket".
// For TCP and UDP, the syntax of addr is "host:port", like "127.0.0.1:8080".
// If host is omitted, as in ":8080", Listen listens on all available interfaces instead of just the interface with the
// given host address.
func NewListener(network, addr string, readTimeout, writeTimeout time.Duration) (net.Listener, error) {
	conn, err := net.Listen(network, addr)
	if err != nil {
		return nil, err
	}

	tl := &Listener{
		Listener:     conn,
		ReadTimeout:  readTimeout,
		WriteTimeout: writeTimeout,
	}
	return tl, nil
}

// NewListenerTLS is just a TLS enabled version of NewListener.
func NewListenerTLS(network, addr, certFile, keyFile string, readTimeout, writeTimeout time.Duration) (net.Listener, error) {
	config := &tls.Config{}
	if config.NextProtos == nil {
		config.NextProtos = []string{"http/1.1"}
	}

	var err error
	config.Certificates = make([]tls.Certificate, 1)
	config.Certificates[0], err = tls.LoadX509KeyPair(certFile, keyFile)
	if err != nil {
		return nil, err
	}

	conn, err := net.Listen(network, addr)
	if err != nil {
		return nil, err
	}
	tl := &Listener{
		Listener:     tls.NewListener(conn, config),
		ReadTimeout:  readTimeout,
		WriteTimeout: writeTimeout,
	}
	return tl, nil
}
