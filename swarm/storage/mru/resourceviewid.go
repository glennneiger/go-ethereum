package mru

import (
	"encoding/json"
	"fmt"
	"hash"
	"net/url"
	"strconv"
	"unsafe"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
)

// ResourceViewID represents a particular user's view of a resource ID
type ResourceViewID struct {
	resourceID ResourceID     `json:"resourceId"`
	ownerAddr  common.Address `json:"ownerAddr"`
}

// ResourceViewID layout:
// ResourceIDLength bytes
// ownerAddr common.AddressLength bytes
const resourceViewIDLength = ResourceIDLength + common.AddressLength

func NewViewID(resourceID *ResourceID, owner common.Address) *ResourceViewID {
	return &ResourceViewID{
		resourceID: *resourceID,
		ownerAddr:  owner,
	}
}

// mapKey calculates a unique id for this view for the cache map in `Handler`
func (u *ResourceViewID) mapKey() uint64 {
	serializedData := make([]byte, resourceViewIDLength)
	u.binaryPut(serializedData)
	hasher := hashPool.Get().(hash.Hash)
	defer hashPool.Put(hasher)
	hasher.Reset()
	hasher.Write(serializedData)
	hash := hasher.Sum(nil)
	return *(*uint64)(unsafe.Pointer(&hash[0]))
}

// binaryPut serializes this ResourceViewID instance into the provided slice
func (u *ResourceViewID) binaryPut(serializedData []byte) error {
	if len(serializedData) != resourceViewIDLength {
		return NewErrorf(ErrInvalidValue, "Incorrect slice size to serialize ResourceViewID. Expected %d, got %d", resourceViewIDLength, len(serializedData))
	}
	var cursor int
	if err := u.resourceID.binaryPut(serializedData[cursor : cursor+ResourceIDLength]); err != nil {
		return err
	}
	cursor += ResourceIDLength

	copy(serializedData[cursor:cursor+common.AddressLength], u.ownerAddr[:])
	cursor += common.AddressLength

	return nil
}

// binaryLength returns the expected size of this structure when serialized
func (u *ResourceViewID) binaryLength() int {
	return resourceViewIDLength
}

// binaryGet restores the current instance from the information contained in the passed slice
func (u *ResourceViewID) binaryGet(serializedData []byte) error {
	if len(serializedData) != resourceViewIDLength {
		return NewErrorf(ErrInvalidValue, "Incorrect slice size to read ResourceViewID. Expected %d, got %d", resourceViewIDLength, len(serializedData))
	}

	var cursor int
	if err := u.resourceID.binaryGet(serializedData[cursor : cursor+ResourceIDLength]); err != nil {
		return err
	}
	cursor += ResourceIDLength

	copy(u.ownerAddr[:], serializedData[cursor:cursor+common.AddressLength])
	cursor += common.AddressLength

	return nil
}

func (u *ResourceViewID) Hex() string {
	serializedData := make([]byte, resourceViewIDLength)
	u.binaryPut(serializedData)
	return hexutil.Encode(serializedData)
}

type resourceViewIDJSON struct {
	ResourceID ResourceID     `json:"resourceId"`
	OwnerAddr  common.Address `json:"ownerAddr"`
}

func (u *ResourceViewID) UnmarshalJSON(data []byte) error {
	var j resourceViewIDJSON
	if err := json.Unmarshal(data, &j); err != nil {
		return err
	}
	u.resourceID = j.ResourceID
	u.ownerAddr = j.OwnerAddr
	return nil
}

func (u *ResourceViewID) MarshalJSON() ([]byte, error) {
	return json.Marshal(&resourceViewIDJSON{
		ResourceID: u.resourceID,
		OwnerAddr:  u.ownerAddr,
	})
}

func (u *ResourceViewID) FromURL(url *url.URL) error {
	query := url.Query()
	startTime, err := strconv.ParseUint(query.Get("starttime"), 10, 64)
	if err != nil {
		return err
	}
	frequency, err := strconv.ParseUint(query.Get("frequency"), 10, 64)
	if err != nil {
		return err
	}
	if err = u.resourceID.Topic.FromHex(query.Get("topic")); err != nil {
		return err
	}
	u.ownerAddr = common.HexToAddress(query.Get("owner"))
	u.resourceID.Frequency = frequency
	u.resourceID.StartTime.Time = startTime
	return nil
}

func (u *ResourceViewID) ToURL(url *url.URL) {
	query := url.Query()
	query.Set("starttime", fmt.Sprintf("%d", u.resourceID.StartTime.Time))
	query.Set("frequency", fmt.Sprintf("%d", u.resourceID.Frequency))
	query.Set("topic", u.resourceID.Topic.Hex())
	query.Set("owner", u.ownerAddr.Hex())
	url.RawQuery = query.Encode()
}
