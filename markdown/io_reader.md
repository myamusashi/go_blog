---
Title: Io golang
Description: test posts
Date: 2024-08-23
Parent: What is this a subpage of
Slug: io_reader
Order: 3
MetaPropertyTitle: This is just check post
MetaDescription: Hellow, cek postingan pertama saya disini
MetaOgURL: https://blog.myamusahi.my.id/posts/io_reader

author:
    name: "Gilang Ramadhan"
    email: "gilang@gmail.com"
---

## Go's io.Reader

Go's io.Reader is defined as:


```go
type Reader interface {
    Read(p []byte) (n int, err error)
}
```
