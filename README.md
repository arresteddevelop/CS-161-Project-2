Project 2 Design
Data Structures
Users
userdata.Username_hash stored as a hash for simplicity, storing password
wasn’t needed.
userdata.FilenameKey is for access to the user’s entire namespace,
deterministically hashed with filename strings
userdata.DecKey, userdata.SignKey RSA Private Keys for use in sharing,
corresponding public keys stored in KeyStore with username and kind. User
objects are never changed after InitUser, so multiple User objects on
different devices do not see different information.
Files
Struct FileChunk struct is the core file data structure. It contains
chunk.Content of bytes, and a Datastore uuid chunk.Prev that acts as a
pointer to the previous chunk, so the file itself forms a pseudo linked list
(in reverse order) between chunks Datastore.
Struct UserFileNode one per file in a user’s namespace.
node.LastChunkUuid corresponds to the uuid containing the uuid of the last
Chunk of the file can be found. This way, the location of the last chunk,
which changes as file grows, is only stored in one place, but every user that
can access the file can retrieve it.
node.FileKey is used to encrypt/decrypt the File chunks.
NodeKey one for each UserFileNode, used for locating & encrypting/decrypting
UserFileNodes. NodeKeys are stored and encrypted using a user’s FilenameKey
hashed with filename strings, and are sent as invitations in
createInvitation.
ShareMap one per file in a user’s namespace. Maps the username of a recipient
to the NodeKey for their UserFileNode for a file, encrypted and stored with
the same NodeKey as corresponding UserFileNode. Every sender therefore has
access to the UserFileNodes and ShareMaps of everyone they’ve shared with,
necessary for revocation.
Helper Functions
DeriveKeys(), DeriveUuid() using HashKDF to get symmetric Keys/MAC Keys for
encryption and deterministic uuids from strings to avoid having to store
them.
AuthEnc(), AuthDec() Encrypt-then-HMAC used on every piece of data in
implementation except invitations.
StoreAuthEnc() encrypting + uploading arbitrary piece of data into Datastore.
Separate routines/functions GetNodeKey(), GetNode(), GetSharedTo(),
GetLastChunk() are used for downloading because differentiating between
errors (e.g, something is missing from datastore) is necessary in some cases.
ChangeAccess() recursive helper function for revoking. Iterates through a
shareMap and updates all the UserFileNodes for users that haven’t been
revoked, then calls on their shareMaps.
Client API
InitUser, GetUser
use Arg2onKey with password and username as salt to determine a key for
encrypting/decrypting userdata.
StoreFile
1. Check if filename already exists in user’s namespace by checking
Datastore for a NodeKey for this file
2. If so, retrieve existing UserFileNode
3. If not, create and store new UserFileNode and ShareMap for the new
File.
4. Create New chunk with content and chunk.Prev set to uuid.Nil. This
ensures that the file is overwritten, as locations of preceding file
chunks are lost.
5. Update LastChunk
LoadFile
Retrieve current LastChunk, then backwards traverse chunks until reaching
uuid.Nil, assembling file contents into a single slice.
AppendToFile
1. Retrieve NodeKey, UserFileNode, and uuid of LastChunk
2. Create a new FileChunk with content, set chunk.Prev = LastChunk
3. Store Chunk and update LastChunk
Downloads: NodeKey, UserFileNode, uuid of Last Chunk
Uploads: appended Chunk, updated uuid of Last Chunk.
Every piece of data except content is a fixed size (uuid or key), so append
scales with size of content only. Worth noting that 64byte HMAC tags on all
pieces of encrypted data contribute significantly to (still constant) Append
overhead.
CreateInvitation
1. Retrieve recipient publicKey from Keystore, serves also to check that
recipient exists
2. Create new UserFileNode (copying fields of sender’s UserFileNode) and
generate corresponding recipient NodeKey. Encrypt and store new
UserFileNode
3. Encrypt and Sign the recipient’s NodeKey, which serves as invitation
4. Get sender’s NodeKey, then SharedTo map
5. Update sharedTo with [recipient username] = recipient NodeKey and store
again
AcceptInvitation
1. Retrieve sender’s VerifyKey from Keystore, checking if sender exists
2. Verify and Decrypt NodeKey from invitationPtr. Delete invitationPtr
from Datastore
3. Use userdata.FilenameKey and filename to get a deterministic uuid for
storing NodeKey. Also serves to allow the recipient to choose a
different filename than sender with no extra steps for either user.
RevokeAccess
1. Retrieve NodeKey, Node, LastChunk, and SharedToMap. Check that the
revoked user has actually been shared with.
2. Delete the LastChunk from Datastore, this is important for making sure
revoked users trigger an error when trying to Append.
3. Iterate through file chunks in reverse order, appending the file into
one content slice, and deleting chunks from the Datastore afterwards.
4. Make a new single chunk for the file. Generate new lastChunkUuid and
new FileKey, encrypt and store the new Chunk with this FileKey.
5. ChangeAccess: iterate through all non-revoked users in sharedTo, and
update their node.LastChunkUuid and node.FileKey so they still have
access to the file. Recurse through SharedMaps to follow sharing
dependencies
Calling AcceptInvitation again won’t help a revoked user get access to the
file, an invitation is just a NodeKey which the revoked user already has.
Without knowing the new location of the File in Datastore, and without the
new FileKey to decrypt the chunks, it is impossible to continue to access the
file.
