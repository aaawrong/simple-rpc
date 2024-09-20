package codec

import (
	"bufio"
	"encoding/json"
	"io"
	"log"
)

type JsonCodec struct {
	conn io.ReadWriteCloser
	buf  *bufio.Writer //bufio 包实现了缓存IO
	dec  *json.Decoder
	enc  *json.Encoder
}

func (c *JsonCodec) Close() error {
	return c.conn.Close()
}

func (c *JsonCodec) ReadHeader(h *Header) error {
	return c.dec.Decode(h)
}

func (c *JsonCodec) ReadBody(i interface{}) error {
	return c.dec.Decode(i)
}

func (c *JsonCodec) Write(h *Header, body interface{}) (err error) {
	defer func() {
		_ = c.buf.Flush()
		if err != nil {
			_ = c.Close()
		}
	}()
	if err = c.enc.Encode(h); err != nil {
		log.Println("rpc codec: gob error encoding header:", err)
		return err
	}
	if err = c.enc.Encode(body); err != nil {
		log.Println("rpc codec: gob error encoding body:", err)
		return err
	}
	return nil
}

var _ Codec = (*JsonCodec)(nil) //编译时检查JsonCodec是否实现了Codec接口

func NewJsonCodec(conn io.ReadWriteCloser) Codec {
	//bufio.Writer 会先将要写入的数据存储在内存的缓冲区中，
	//等到缓冲区满了或者显式调用了 Flush() 时，才将缓冲区的内容一次性写入底层的 conn
	buf := bufio.NewWriter(conn)
	return &JsonCodec{
		conn: conn,
		buf:  buf,
		dec:  json.NewDecoder(conn),
		enc:  json.NewEncoder(buf),
	}
}
