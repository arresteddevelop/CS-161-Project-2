package client

// CS 161 Project 2

// Only the following imports are allowed! ANY additional imports
// may break the autograder!
// - bytes
// - encoding/hex
// - encoding/json
// - errors
// - fmt
// - github.com/cs161-staff/project2-userlib
// - github.com/google/uuid
// - strconv
// - strings

import (
	"encoding/json"

	userlib "github.com/cs161-staff/project2-userlib"
	"github.com/google/uuid"

	// hex.EncodeToString(...) is useful for converting []byte to string

	// Useful for string manipulation

	// Useful for formatting strings (e.g. `fmt.Sprintf`).
	"fmt"

	// Useful for creating new error messages to return using errors.New("...")
	"errors"

	// Optional.
	_ "strconv"
)

// This serves two purposes: it shows you a few useful primitives,
// and suppresses warnings for imports not being used. It can be
// safely deleted!
func someUsefulThings() {

	// Creates a random UUID.
	randomUUID := uuid.New()

	// Prints the UUID as a string. %v prints the value in a default format.
	// See https://pkg.go.dev/fmt#hdr-Printing for all Golang format string flags.
	userlib.DebugMsg("Random UUID: %v", randomUUID.String())

	// Creates a UUID deterministically, from a sequence of bytes.
	hash := userlib.Hash([]byte("user-structs/alice"))
	deterministicUUID, err := uuid.FromBytes(hash[:16])
	if err != nil {
		// Normally, we would `return err` here. But, since this function doesn't return anything,
		// we can just panic to terminate execution. ALWAYS, ALWAYS, ALWAYS check for errors! Your
		// code should have hundreds of "if err != nil { return err }" statements by the end of this
		// project. You probably want to avoid using panic statements in your own code.
		panic(errors.New("An error occurred while generating a UUID: " + err.Error()))
	}
	userlib.DebugMsg("Deterministic UUID: %v", deterministicUUID.String())

	// Declares a Course struct type, creates an instance of it, and marshals it into JSON.
	type Course struct {
		name      string
		professor []byte
	}

	course := Course{"CS 161", []byte("Nicholas Weaver")}
	courseBytes, err := json.Marshal(course)
	if err != nil {
		panic(err)
	}

	userlib.DebugMsg("Struct: %v", course)
	userlib.DebugMsg("JSON Data: %v", courseBytes)

	// Generate a random private/public keypair.
	// The "_" indicates that we don't check for the error case here.
	var pk userlib.PKEEncKey
	var sk userlib.PKEDecKey
	pk, sk, _ = userlib.PKEKeyGen()
	userlib.DebugMsg("PKE Key Pair: (%v, %v)", pk, sk)

	// Here's an example of how to use HBKDF to generate a new key from an input key.
	// Tip: generate a new key everywhere you possibly can! It's easier to generate new keys on the fly
	// instead of trying to think about all of the ways a key reuse attack could be performed. It's also easier to
	// store one key and derive multiple keys from that one key, rather than
	originalKey := userlib.RandomBytes(16)
	derivedKey, err := userlib.HashKDF(originalKey, []byte("mac-key"))
	if err != nil {
		panic(err)
	}
	userlib.DebugMsg("Original Key: %v", originalKey)
	userlib.DebugMsg("Derived Key: %v", derivedKey)

	// A couple of tips on converting between string and []byte:
	// To convert from string to []byte, use []byte("some-string-here")
	// To convert from []byte to string for debugging, use fmt.Sprintf("hello world: %s", some_byte_arr).
	// To convert from []byte to string for use in a hashmap, use hex.EncodeToString(some_byte_arr).
	// When frequently converting between []byte and string, just marshal and unmarshal the data.
	//
	// Read more: https://go.dev/blog/strings

	// Here's an example of string interpolation!
	_ = fmt.Sprintf("%s_%d", "file", 1)
}

// This is the type definition for the User struct.
// A Go struct is like a Python or Java class - it can have attributes
// (e.g. like the Username attribute) and methods (e.g. like the StoreFile method below).
type User struct {
	Username_hash []byte
	FilenameKey   []byte
	DecKey        userlib.PKEDecKey
	SignKey       userlib.DSSignKey

	// You can add other attributes here if you want! But note that in order for attributes to
	// be included when this struct is serialized to/from JSON, they must be capitalized.
	// On the flipside, if you have an attribute that you want to be able to access from
	// this struct's methods, but you DON'T want that value to be included in the serialized value
	// of this struct that's stored in datastore, then you can use a "private" variable (e.g. one that
	// begins with a lowercase letter).
}

type UserFileNode struct {
	LastChunkUuid uuid.UUID //should never change unless revoked
	FileKey       []byte
}

type FileChunk struct {
	Content []byte
	Prev    uuid.UUID
}

type ShareMap map[string][]byte

func DeriveKeys(sourceKey []byte) (symKey []byte, macKey []byte, err error) {
	symKey, err = userlib.HashKDF(sourceKey, []byte("SymEnc"))
	if err != nil {
		return nil, nil, err
	}
	symKey = symKey[:16]

	macKey, err = userlib.HashKDF(sourceKey, []byte("HMAC"))
	if err != nil {
		return nil, nil, err
	}
	macKey = macKey[:16]

	return symKey, macKey, nil
}

func DeriveUuid(sourceKey []byte, purpose string) (u uuid.UUID, err error) {
	uBytes, err := userlib.HashKDF(sourceKey, []byte(purpose))
	if err != nil {
		return uuid.Nil, err
	}

	u, err = uuid.FromBytes(uBytes[:16])
	if err != nil {
		return uuid.Nil, err
	}
	return u, nil
}

func AuthEnc(sourceKey []byte, unencrypted []byte) (storable []byte, err error) {

	symKey, macKey, err := DeriveKeys(sourceKey)
	if err != nil {
		return nil, err
	}

	iv := userlib.RandomBytes(16)
	bytes_enc := userlib.SymEnc(symKey, iv, unencrypted)
	hmac, err := userlib.HMACEval(macKey, bytes_enc)
	storable = append(hmac, bytes_enc...)
	return storable, nil
}

func AuthDec(sourceKey []byte, stored []byte) (unencrypted []byte, err error) {

	symKey, macKey, err := DeriveKeys(sourceKey)
	if err != nil {
		return nil, err
	}

	hmac1 := stored[:64]
	bytes_enc := stored[64:]
	hmac2, err := userlib.HMACEval(macKey, bytes_enc)
	if err != nil {
		return nil, err
	}

	if !userlib.HMACEqual(hmac2, hmac1) {
		return nil, errors.New("HMAC fail")
	}

	unencrypted = userlib.SymDec(symKey, bytes_enc)
	return unencrypted, nil

}

func StoreAuthEnc(data any, key []byte, dataUuid uuid.UUID) (err error) {

	dataBytes, err := json.Marshal(data)
	if err != nil {
		return err
	}

	storable_data, err := AuthEnc(key, dataBytes)
	if err != nil {
		return err
	}

	userlib.DatastoreSet(dataUuid, storable_data)
	return
}

///////////////////////////////////End Helper Functions /////////////////////////////////

func InitUser(username string, password string) (userdataptr *User, err error) {
	var userdata User
	var publicKey userlib.PKEEncKey
	var verifyKey userlib.DSVerifyKey

	userdata.Username_hash = userlib.Hash([]byte(username))

	userdata.FilenameKey = userlib.RandomBytes(16)

	publicKey, userdata.DecKey, _ = userlib.PKEKeyGen()
	userlib.KeystoreSet(username+"_Public", publicKey)

	userdata.SignKey, verifyKey, _ = userlib.DSKeyGen()
	userlib.KeystoreSet(username+"_Verify", verifyKey)

	userKey := userlib.Argon2Key([]byte(password), userdata.Username_hash, 16)

	userUuid, err := DeriveUuid(userKey, "UUID")
	if err != nil {
		return nil, err
	}

	StoreAuthEnc(userdata, userKey, userUuid)

	return &userdata, nil
}

func GetUser(username string, password string) (userdataptr *User, err error) {
	var userdata User

	username_hash := userlib.Hash([]byte(username))

	userKey := userlib.Argon2Key([]byte(password), username_hash, 16)

	userUuid, err := DeriveUuid(userKey, "UUID")
	if err != nil {
		return nil, err
	}

	stored_userdata, ok := userlib.DatastoreGet(userUuid)
	if !ok {
		return nil, errors.New("No such user " + username)
	}

	userBytes, err := AuthDec(userKey, stored_userdata)
	if err != nil {
		return nil, err
	}

	err = json.Unmarshal(userBytes, &userdata)
	if err != nil {
		return nil, err
	}

	userdataptr = &userdata
	return userdataptr, nil
}

func (userdata *User) StoreFile(filename string, content []byte) (err error) {
	var chunk FileChunk
	var node UserFileNode
	var NodeKey []byte

	NodeKeyUuid, err := DeriveUuid(userdata.FilenameKey, filename)
	if err != nil {
		return err
	}
	// check if file already exstis
	_, ok := userlib.DatastoreGet(NodeKeyUuid)
	if ok {
		// overwrite
		NodeKey, err = GetNodeKey(filename, userdata.FilenameKey)
		if err != nil {
			return err
		}

		node, err = GetNode(NodeKey)
		if err != nil {
			return err
		}

	} else {
		NodeKey := userlib.RandomBytes(16)
		storable_NodeKey, err := AuthEnc(userdata.FilenameKey, NodeKey)
		if err != nil {
			return err
		}
		userlib.DatastoreSet(NodeKeyUuid, storable_NodeKey)

		node.LastChunkUuid = uuid.New()
		node.FileKey = userlib.RandomBytes(16)

		sharedTo := make(ShareMap)

		sharedToUuid, err := DeriveUuid(NodeKey, "ShareMap")
		if err != nil {
			return err
		}

		err = StoreAuthEnc(sharedTo, NodeKey, sharedToUuid)
		if err != nil {
			return err
		}

		nodeUuid, err := DeriveUuid(NodeKey, "UserFileNode")
		if err != nil {
			return err
		}

		err = StoreAuthEnc(node, NodeKey, nodeUuid)
		if err != nil {
			return err
		}
	}

	chunkUuid := uuid.New()
	chunk.Content = content
	chunk.Prev = uuid.Nil

	err = StoreAuthEnc(chunk, node.FileKey, chunkUuid)
	if err != nil {
		return err
	}

	err = StoreAuthEnc(chunkUuid, node.FileKey, node.LastChunkUuid)
	if err != nil {
		return err
	}

	return nil
}

func GetNodeKey(filename string, filenameKey []byte) (NodeKey []byte, err error) {

	NodeKeyUuid, err := DeriveUuid(filenameKey, filename)
	if err != nil {
		return nil, err
	}

	stored_NodeKey, ok := userlib.DatastoreGet(NodeKeyUuid)
	if !ok {
		return nil, errors.New(filename + "not in users'namespace")
	}

	NodeKey, err = AuthDec(filenameKey, stored_NodeKey)
	if err != nil {
		return nil, err
	}

	return NodeKey, nil

}

func GetNode(NodeKey []byte) (node UserFileNode, err error) {
	// retrieve and decrypt Node

	nodeUuid, err := DeriveUuid(NodeKey, "UserFileNode")
	if err != nil {
		return node, err
	}

	stored_node, ok := userlib.DatastoreGet(nodeUuid)
	if !ok {
		return node, errors.New("Node missing")
	}

	nodeBytes, err := AuthDec(NodeKey, stored_node)
	if err != nil {
		return node, err
	}

	err = json.Unmarshal(nodeBytes, &node)
	if err != nil {
		return node, err
	}

	return node, nil
}

func GetLastChunk(node UserFileNode) (lastChunk uuid.UUID, err error) {

	stored_lastChunk, ok := userlib.DatastoreGet(node.LastChunkUuid)
	if !ok {
		return uuid.Nil, errors.New("lastChunkuuid is gone")
	}

	LastChunkBytes, err := AuthDec(node.FileKey, stored_lastChunk)
	if err != nil {
		return uuid.Nil, err
	}

	err = json.Unmarshal(LastChunkBytes, &lastChunk)
	if err != nil {
		return uuid.Nil, err
	}

	return lastChunk, nil

}

func GetSharedTo(NodeKey []byte) (sharedTo ShareMap, err error) {

	sharedToUuid, err := DeriveUuid(NodeKey, "ShareMap")
	if err != nil {
		return nil, err
	}

	stored_sharedTo, ok := userlib.DatastoreGet(sharedToUuid)
	if !ok {
		return nil, errors.New("Sharemap missing")
	}

	sharedToBytes, err := AuthDec(NodeKey, stored_sharedTo)
	if err != nil {
		return nil, err
	}

	err = json.Unmarshal(sharedToBytes, &sharedTo)
	if err != nil {
		return nil, err
	}
	return sharedTo, nil
}

func (userdata *User) AppendToFile(filename string, content []byte) error {
	var chunk FileChunk

	NodeKey, err := GetNodeKey(filename, userdata.FilenameKey)
	if err != nil {
		return err
	}

	node, err := GetNode(NodeKey)
	if err != nil {
		return err
	}

	lastChunk, err := GetLastChunk(node)
	if err != nil {
		return err
	}

	chunkUuid := uuid.New()
	chunk.Content = content
	chunk.Prev = lastChunk

	err = StoreAuthEnc(chunk, node.FileKey, chunkUuid)
	if err != nil {
		return err
	}

	err = StoreAuthEnc(chunkUuid, node.FileKey, node.LastChunkUuid)
	if err != nil {
		return err
	}

	return nil

}

func (userdata *User) LoadFile(filename string) (content []byte, err error) {
	var chunk FileChunk

	NodeKey, err := GetNodeKey(filename, userdata.FilenameKey)
	if err != nil {
		return nil, err
	}

	node, err := GetNode(NodeKey)
	if err != nil {
		return nil, err
	}

	lastChunk, err := GetLastChunk(node)
	if err != nil {
		return nil, err
	}

	stored_chunk, ok := userlib.DatastoreGet(lastChunk)
	if !ok {
		return nil, errors.New("no chunk")
	}

	chunkBytes, err := AuthDec(node.FileKey, stored_chunk)
	if err != nil {
		return nil, err
	}

	err = json.Unmarshal(chunkBytes, &chunk)
	if err != nil {
		return nil, err
	}

	content = chunk.Content

	for chunk.Prev != uuid.Nil {
		stored_chunk, ok = userlib.DatastoreGet(chunk.Prev)
		if !ok {
			return content, errors.New("missing chunk")
		}

		chunkBytes, err = AuthDec(node.FileKey, stored_chunk)
		if err != nil {
			return content, err
		}

		err = json.Unmarshal(chunkBytes, &chunk)
		if err != nil {
			return content, err
		}
		content = append(chunk.Content, content...)
	}
	return content, nil

}

func (userdata *User) CreateInvitation(filename string, recipientUsername string) (invitationPtr uuid.UUID, err error) {
	var rNode UserFileNode

	rPublicKey, ok := userlib.KeystoreGet(recipientUsername + "_Public")
	if !ok {
		return invitationPtr, errors.New("recpient Doesn't exist")
	}

	// Retrieve sender File Node

	sNodeKey, err := GetNodeKey(filename, userdata.FilenameKey)
	if err != nil {
		return invitationPtr, err
	}

	rNodeKey := userlib.RandomBytes(16)

	invitationPtr = uuid.New()

	// Use RSA to encrypt invitation and Store at invitationPtr

	rNodeKey_enc, err := userlib.PKEEnc(rPublicKey, rNodeKey)
	if err != nil {
		return invitationPtr, err
	}
	rNodeKey_sig, err := userlib.DSSign(userdata.SignKey, rNodeKey_enc)
	if err != nil {
		return invitationPtr, err
	}

	userlib.DatastoreSet(invitationPtr, append(rNodeKey_sig, rNodeKey_enc...))

	sNode, err := GetNode(sNodeKey)
	if err != nil {
		return invitationPtr, err
	}

	rNode.FileKey = sNode.FileKey
	rNode.LastChunkUuid = sNode.LastChunkUuid

	// Store rNode

	rNodeUuid, err := DeriveUuid(rNodeKey, "UserFileNode")
	if err != nil {
		return invitationPtr, err
	}

	err = StoreAuthEnc(rNode, rNodeKey, rNodeUuid)
	if err != nil {
		return invitationPtr, err
	}

	// Store rSharedTo

	rSharedTo := make(ShareMap)

	rSharedToUuid, err := DeriveUuid(rNodeKey, "ShareMap")
	if err != nil {
		return invitationPtr, err
	}

	err = StoreAuthEnc(rSharedTo, rNodeKey, rSharedToUuid)
	if err != nil {
		return invitationPtr, err
	}

	// Update sender shareMap and Restore

	sSharedTo, err := GetSharedTo(sNodeKey)

	sSharedTo[recipientUsername] = rNodeKey

	sSharedToUuid, err := DeriveUuid(sNodeKey, "ShareMap")
	if err != nil {
		return invitationPtr, err
	}

	err = StoreAuthEnc(sSharedTo, sNodeKey, sSharedToUuid)
	if err != nil {
		return invitationPtr, err
	}

	return invitationPtr, nil

}

func (userdata *User) AcceptInvitation(senderUsername string, invitationPtr uuid.UUID, filename string) (err error) {

	NodeKeyUuid, err := DeriveUuid(userdata.FilenameKey, filename)
	if err != nil {
		return err
	}
	_, ok := userlib.DatastoreGet(NodeKeyUuid)
	if ok {
		return errors.New(filename + "already in namespace")
	}

	sVerifyKey, ok := userlib.KeystoreGet(senderUsername + "_Verify")
	if !ok {
		return errors.New("sender doesnt exist")
	}

	data, ok := userlib.DatastoreGet(invitationPtr)
	if !ok {
		return errors.New("no invitationPTr")
	}

	NodeKey_sig := data[:256]
	NodeKey_enc := data[256:]

	err = userlib.DSVerify(sVerifyKey, NodeKey_enc, NodeKey_sig)
	if err != nil {
		return errors.New("DSVerify fail")
	}

	NodeKey, err := userlib.PKEDec(userdata.DecKey, NodeKey_enc)
	if err != nil {
		return errors.New("PKEDec Failure")
	}

	userlib.DatastoreDelete(invitationPtr) // clean pointer

	node, err := GetNode(NodeKey)
	if err != nil {
		return err
	}

	nodeUuid, err := DeriveUuid(NodeKey, "UserFileNode")
	if err != nil {
		return err
	}

	err = StoreAuthEnc(node, NodeKey, nodeUuid)
	if err != nil {
		return err
	}

	storable_NodeKey, err := AuthEnc(userdata.FilenameKey, NodeKey)
	if err != nil {
		return err
	}

	userlib.DatastoreSet(NodeKeyUuid, storable_NodeKey)

	return nil

}

func ChangeAcess(sharedTo ShareMap, revoked string, newChunkUuid uuid.UUID, newFileKey []byte) (err error) {
	for username, NodeKey := range sharedTo {
		nodeUuid, err := DeriveUuid(NodeKey, "UserFileNode")
		if err != nil {
			return err
		}
		if username != revoked {
			node, err := GetNode(NodeKey)
			if err != nil {
				return err
			}

			node.LastChunkUuid = newChunkUuid
			node.FileKey = newFileKey

			err = StoreAuthEnc(node, NodeKey, nodeUuid)
			if err != nil {
				return err
			}

			uSharedTo, err := GetSharedTo(NodeKey)
			if err != nil {
				return err
			}

			err = ChangeAcess(uSharedTo, revoked, newChunkUuid, newFileKey)
			if err != nil {
				return err
			}

		}
	}
	return nil

}

func (userdata *User) RevokeAccess(filename string, recipientUsername string) error {

	NodeKey, err := GetNodeKey(filename, userdata.FilenameKey)
	if err != nil {
		return err
	}

	node, err := GetNode(NodeKey)
	if err != nil {
		return err
	}

	nodeUuid, err := DeriveUuid(NodeKey, "UserFileNode")
	if err != nil {
		return err
	}

	sharedTo, err := GetSharedTo(NodeKey)
	if err != nil {
		return err
	}

	_, ok := sharedTo[recipientUsername]

	if !ok {
		return errors.New(filename + "hasn't been shared with" + recipientUsername)
	}

	lastChunk, err := GetLastChunk(node)
	if err != nil {
		return err
	}

	var chunk FileChunk

	stored_chunk, ok := userlib.DatastoreGet(lastChunk)
	if !ok {
		return errors.New("no chunk")
	}

	userlib.DatastoreDelete(lastChunk)
	userlib.DatastoreDelete(node.LastChunkUuid) //delete lastChunk

	chunkBytes, err := AuthDec(node.FileKey, stored_chunk)
	if err != nil {
		return err
	}

	err = json.Unmarshal(chunkBytes, &chunk)
	if err != nil {
		return err
	}

	content := chunk.Content

	for chunk.Prev != uuid.Nil {

		stored_chunk, ok = userlib.DatastoreGet(chunk.Prev)
		if !ok {
			return errors.New("missing chunk")
		}

		userlib.DatastoreDelete(chunk.Prev) // Delete each chunk

		chunkBytes, err = AuthDec(node.FileKey, stored_chunk)
		if err != nil {
			return err
		}

		err = json.Unmarshal(chunkBytes, &chunk)
		if err != nil {
			return err
		}
		content = append(chunk.Content, content...)
	}

	node.FileKey = userlib.RandomBytes(16)
	node.LastChunkUuid = uuid.New()

	chunkUuid := uuid.New()
	chunk.Content = content
	chunk.Prev = uuid.Nil

	err = StoreAuthEnc(chunk, node.FileKey, chunkUuid)
	if err != nil {
		return err
	}

	err = StoreAuthEnc(chunkUuid, node.FileKey, node.LastChunkUuid)
	if err != nil {
		return err
	}

	err = StoreAuthEnc(node, NodeKey, nodeUuid)
	if err != nil {
		return err
	}

	err = ChangeAcess(sharedTo, recipientUsername, node.LastChunkUuid, node.FileKey)
	if err != nil {
		return err
	}

	return nil

}
