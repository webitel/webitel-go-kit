package fts_client

const exchange = "fts-stock"

type Publisher interface {
	Send(exchange string, rk string, body []byte) error
}

type Client struct {
	publisher Publisher
}

func New(p Publisher) *Client {
	return &Client{
		publisher: p,
	}
}

func (c *Client) Create(domainId int64, objectName string, id any, row any) error {
	msg, err := NewMessageJSON(domainId, objectName, id, row)
	if err != nil {
		return err
	}

	return c.publisher.Send(exchange, MessageCreate, msg)
}

func (c *Client) Update(domainId int64, objectName string, id any, row any) error {
	msg, err := NewMessageJSON(domainId, objectName, id, row)
	if err != nil {
		return err
	}

	return c.publisher.Send(exchange, MessageUpdate, msg)
}

func (c *Client) Delete(domainId int64, objectName string, id any) error {
	msg, err := NewMessageJSON(domainId, objectName, id, nil)
	if err != nil {
		return err
	}

	return c.publisher.Send(exchange, MessageDelete, msg)
}
