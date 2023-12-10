package main

import (
	"log"
	"encoding/hex"
	"os"
	"fmt"
	"bytes"

	bolt "go.etcd.io/bbolt"
)

// moveBucket moves the inner bucket with key 'oldkey' to a new bucket with key 'newkey'
// must be used within an Update-transaction
func moveBucket(oldParent, newParent *bolt.Bucket, oldkey, newkey []byte) error {
    oldBuck := oldParent.Bucket(oldkey)
    newBuck, err := newParent.CreateBucket(newkey)
    if err != nil {
        return err
    }

    err = oldBuck.ForEach(func(k, v []byte) error {
        if v == nil {
            // Nested bucket
            return moveBucket(oldBuck, newBuck, k, k)
        } else {
            // Regular value
            return newBuck.Put(k, v)
        }
    })
    if err != nil {
        return err
    }

    return oldParent.DeleteBucket(oldkey)
}

func replaceAllString(parent *bolt.Bucket, oldstr []byte, newstr []byte) error {
    err := parent.ForEach(func(k, v []byte) error {
        if v == nil {
            // Nested bucket
            if bytes.Contains(k, oldstr) {
                fmt.Printf("found needle in bucket: %#v\n", k)
                newkey := make([]byte, len(k))
                copy(newkey, k)
                oldstr_offset := bytes.Index(k, oldstr)
                copy(newkey[oldstr_offset:], newstr)
                moveBucket(parent, parent, k, newkey)
                return replaceAllString(parent.Bucket(newkey), oldstr, newstr)
            } else {
                return replaceAllString(parent.Bucket(k), oldstr, newstr)
            }
        } else {
            // Regular value
            if bytes.Contains(v, oldstr) {
                fmt.Printf("found needle in k: %#v    v: %#v\n", k, v)
                newval := make([]byte, len(v))
                copy(newval, v)
                oldstr_offset := bytes.Index(v, oldstr)
                copy(newval[oldstr_offset:], newstr)
                parent.Put(k, newval)
            }

            if bytes.Contains(k, oldstr) {
                fmt.Printf("found needle in k: %#v    v: %#v\n", k, v)
                newkey := make([]byte, len(k))
                copy(newkey, k)
                oldstr_offset := bytes.Index(k, oldstr)
                copy(newkey[oldstr_offset:], newstr)
                newval := parent.Get(k)
                parent.Put(newkey, newval)
                parent.Delete(k)
            }

            return nil
        }
    })
    if err != nil {
        return err
    }

    return nil
}

func main() {
	channel_db := os.Args[1]
	old_pubkey, err := hex.DecodeString(os.Args[2])
	if err != nil {
		log.Fatal(err)
	}

	new_pubkey, err := hex.DecodeString(os.Args[3])
	if err != nil {
		log.Fatal(err)
	}

	db, err := bolt.Open(channel_db, 0600, nil)
	if err != nil {
		log.Fatal(err)
	}
	defer db.Close()

	err = db.Update(func(tx *bolt.Tx) error {
		tx.ForEach(func(name []byte, b *bolt.Bucket) error {
			replaceAllString(b, old_pubkey, new_pubkey)
			return nil
		})
		return nil
	})
}
