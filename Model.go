package goboots

import (
	"labix.org/v2/mgo/bson"
	"log"
	"reflect"
	"time"
)

const (
	QueryOpEq                  = "$eq"
	QueryOpLessOrEq            = "$lte"
	QueryOpGreaterOrEq         = "$gte"
	QueryOpLess                = "$lt"
	QueryOpGreater             = "$gt"
	QueryOpNotEqual            = "$ne"
	QueryOpInsideSlice         = "$in"
	QueryOpContainsAllElements = "$all"
	QueryOpNotContainsElements = "$nin"
)

var (
	//typeBinary         = reflect.TypeOf(Binary{})
	//typeObjectId       = reflect.TypeOf(ObjectId(""))
	//typeSymbol         = reflect.TypeOf(Symbol(""))
	//typeMongoTimestamp = reflect.TypeOf(MongoTimestamp(0))
	//typeOrderKey       = reflect.TypeOf(MinKey)
	//typeDocElem        = reflect.TypeOf(DocElem{})
	//typeRaw            = reflect.TypeOf(Raw{})
	//typeURL            = reflect.TypeOf(url.URL{})
	typeTime = reflect.TypeOf(time.Time{})
)

type M2 bson.M

type IModel interface {
	GetId() bson.ObjectId
	GetCollectionName() string
}

type Model struct {
	Id bson.ObjectId `bson:"_id,omitempty" json:"id"`
}

func (m *Model) GetId() bson.ObjectId {
	return m.Id
}

func (m *Model) SetIdHex(idHex string) error {
	if !bson.IsObjectIdHex(idHex) {
		return &AppError{-1, "ObjectId Hex '" + idHex + "' is not valid!"}
	}
	m.Id = bson.ObjectIdHex(idHex)
	return nil
}

func (m *Model) GenerateId() {
	m.Id = bson.NewObjectId()
}

func (m *Model) RemoveById() {
	log.Println("catson"+reflect.TypeOf(m).Name(), reflect.TypeOf(m))
}

func (m *Model) FindOne(input interface{}, output IModel, omitEmpty bool) error {
	bs, _ := bson.Marshal(input)
	mp := bson.M{}
	bson.Unmarshal(bs, mp)
	if omitEmpty {
		for k, v := range mp {
			vv := reflect.ValueOf(v)
			if isZero(vv) {
				delete(mp, k)
			}
		}
	}
	//return DB.C(output.GetCollectionName()).Find(mp).One(output)
	//TODO: remove this dependency
	return nil
}

func (m *Model) Query(input IModel, omitEmpty bool) *M2 {
	bs, _ := bson.Marshal(input)
	mp := M2{}
	bson.Unmarshal(bs, mp)
	if omitEmpty {
		for k, v := range mp {
			vv := reflect.ValueOf(v)
			if isZero(vv) {
				delete(mp, k)
			}
		}
	}
	mp["_8cname9__"] = input.GetCollectionName()
	return &mp
}

func (m *M2) Where(name string, signal string, value interface{}) *M2 {
	mm := *m
	mm[name] = bson.M{signal: value}
	return &mm
}

func (m *M2) One(result IModel) error {
	mm := *m
	//cname, _ := mm["_8cname9__"].(string)
	delete(mm, "_8cname9__")
	//return DB.C(cname).Find(mm).One(result)
	//TODO: remove dependency
	return nil
}

func (m *M2) All(result interface{}) error {
	mm := *m
	//cname, _ := mm["_8cname9__"].(string)
	delete(mm, "_8cname9__")
	//return DB.C(cname).Find(mm).All(result)
	//TODO: remove dependency
	return nil
}

func isZero(v reflect.Value) bool {
	switch v.Kind() {
	case reflect.String:
		return len(v.String()) == 0
	case reflect.Ptr, reflect.Interface:
		return v.IsNil()
	case reflect.Slice:
		return v.Len() == 0
	case reflect.Map:
		return v.Len() == 0
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		return v.Int() == 0
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64, reflect.Uintptr:
		return v.Uint() == 0
	case reflect.Float32, reflect.Float64:
		return v.Float() == 0
	case reflect.Bool:
		return !v.Bool()
	case reflect.Struct:
		if v.Type() == typeTime {
			return v.Interface().(time.Time).IsZero()
		}
	}
	return false
}
