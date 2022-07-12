package chunk

import "reflect"

// Merge recursively merges the src and dst maps. Key conflicts are resolved by
// preferring src, or recursively descending, if both src and dst are maps.
// copy and edit from https://github.com/peterbourgon/mergemap/blob/master/mergemap.go
// BSD-2-Clause license
func tomerge(dst, src map[string]interface{}, depth int) map[string]interface{} {
	if depth > 32 {
		panic("too deep!")
	}
	for key, srcVal := range src {
		if dstVal, ok := dst[key]; ok {
			srcMap, srcMapOk := mapify(srcVal)
			dstMap, dstMapOk := mapify(dstVal)
			if srcMapOk && dstMapOk {
				srcVal = tomerge(dstMap, srcMap, depth+1)
			} else {
				srcVal = sliceDoSome(dstVal, srcVal, depth)
			}
		}
		dst[key] = srcVal
	}
	return dst
}

func mapify(i interface{}) (map[string]interface{}, bool) {
	value := reflect.ValueOf(i)
	if value.Kind() == reflect.Map {
		m := map[string]interface{}{}
		for _, k := range value.MapKeys() {
			m[k.String()] = value.MapIndex(k).Interface()
		}
		return m, true
	}
	return map[string]interface{}{}, false
}

func sliceDoSome(dst, src any, depth int) any {
	value := reflect.ValueOf(dst)
	if value.Kind() != reflect.Slice {
		return src
	}

	dt := reflect.TypeOf(dst)
	ek := dt.Elem().Kind()
	if ek == reflect.Int32 || ek == reflect.Int || ek == reflect.Int64 {
		len := value.Len()
		nl := reflect.MakeSlice(dt, len, len)
		sv := reflect.ValueOf(src)
		for i := 0; i < len; i++ {
			rv := sv.Index(i)
			switch ek {
			case reflect.Int32:
				rv = reflect.ValueOf(int32(rv.Elem().Float()))
			case reflect.Int:
				rv = reflect.ValueOf(int(rv.Elem().Float()))
			case reflect.Int64:
				rv = reflect.ValueOf(int64(rv.Elem().Float()))
			}
			nl.Index(i).Set(rv)
		}
		return nl.Interface()
	}

	if ek != reflect.Interface {
		return src
	}
	dstl := dst.([]any)
	srcl, ok := src.([]any)
	if !ok {
		return dst
	}
	for i, dv := range dstl {
		srcMap, srcMapOk := mapify(srcl[i])
		dstMap, dstMapOk := mapify(dv)
		if srcMapOk && dstMapOk {
			dstl[i] = tomerge(dstMap, srcMap, depth+1)
		} else {
			dstl[i] = sliceDoSome(dv, srcl[i], depth)
		}
	}
	return dstl

}
