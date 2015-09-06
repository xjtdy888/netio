package store
type Store interface  {

    Set(key, val string)

    Get(key string) string

    Has(key string) bool

    Del(key string) 

}
