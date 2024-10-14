package codec

import (
	"bufio"
	"encoding/json"
	"io"
	"log"
)

type JsonCodec struct {
	conn io.ReadWriteCloser //conn 是由构建函数传入，通常是通过 TCP 或者 Unix 建立 socket 时得到的链接实例
	buf  *bufio.Writer      //bufio 包实现了缓存IO，buf 是为了防止阻塞而创建的带缓冲的 Writer
	dec  *json.Decoder
	enc  *json.Encoder
}

var _ Codec = (*JsonCodec)(nil) //编译时检查JsonCodec是否实现了Codec接口

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

func NewJsonCodec(conn io.ReadWriteCloser) Codec {
	// bufio.Writer 会先将要写入的数据存储在内存的缓冲区中，
	// 等到缓冲区满了或者显式调用了 Flush() 时，才将缓冲区的内容一次性写入底层的 conn
	buf := bufio.NewWriter(conn)
	return &JsonCodec{
		conn: conn,
		buf:  buf,
		dec:  json.NewDecoder(conn),
		enc:  json.NewEncoder(buf),
	}
}
