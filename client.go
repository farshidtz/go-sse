package net

import (
	"errors"
	"net/http"
)

type Client struct {
	id             string
	responseWriter http.ResponseWriter
	request        *http.Request
	flusher        http.Flusher

	msg      chan []byte
	doneChan chan interface{}
}

func NewClient(id string, w http.ResponseWriter, r *http.Request) (*Client, error) {
	flusher, ok := w.(http.Flusher)
	if !ok {
		http.Error(w, "Streaming unsupported!", http.StatusInternalServerError)
		return nil, errors.New("streaming unsupported")
	}

	return &Client{
		id:             id,
		responseWriter: w,
		request:        r,
		flusher:        flusher,
		msg:            make(chan []byte),
		doneChan:       make(chan interface{}, 1),
	}, nil
}

func (c *Client) Id() string {
	return c.id
}

func (c *Client) Send(event Event) {
	c.msg <- event.Prepare()
}

func (c *Client) serve(onClose func()) {
writeLoop:
	for {
		select {
		case <-c.request.Context().Done():
			break writeLoop
		case msg, open := <-c.msg:
			if !open {
				break writeLoop
			}
			_, err := c.responseWriter.Write(msg)
			if err != nil {
				return
			}
			c.flusher.Flush()
		}
	}

	c.doneChan <- true
	onClose()
}

func (c *Client) Done() <-chan interface{} {
	return c.doneChan
}
