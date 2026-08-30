package engine

// persistent_containers.go — durable room containers (e.g. player home chests).
//
// Room containers normally live only in the in-memory roomContainerContents map
// (see containers.go) and are lost on every restart, along with any runtime
// ItemBits changes on the room item itself, since e.rooms is rebuilt fresh from
// the .SCR scripts on every boot.
//
// ItemBit 19 is reserved as a marker: any room container whose live ItemBits has
// that bit set is treated as persistent. A script flags a specific chest instance
// with "EQUAL ITEMBIT19 1" (e.g. from an IFENTRY block the first time a player
// claims their home), which both marks it in memory and — via the hook in
// scripts.go's ITEMBIT setVar handler — registers it in MongoDB so the flag
// itself survives a restart, not just the contents. From then on, every PUT/GET
// against that container (via roomContainerSet/roomContainerDelete) writes its
// current contents through to the same record.

import (
	"context"
	"log"

	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo/options"
)

const persistentContainerBit = 19

const persistentContainersCollection = "persistentContainers"

// persistentContainerDoc is the MongoDB record for one durable room container,
// keyed by the same "<roomNumber>:<itemRef>" string containerKey() already uses
// for the in-memory map.
type persistentContainerDoc struct {
	ID         string          `bson:"_id"`
	RoomNumber int             `bson:"roomNumber"`
	ItemRef    int             `bson:"itemRef"`
	Contents   []InventoryItem `bson:"contents"`
}

// isPersistentContainerRef reports whether the room item currently sitting at
// roomNum:ref has the persistent-container bit set.
func (e *GameEngine) isPersistentContainerRef(roomNum, ref int) bool {
	room := e.rooms[roomNum]
	if room == nil {
		return false
	}
	for i := range room.Items {
		if room.Items[i].Ref == ref {
			return room.Items[i].ItemBits&(1<<persistentContainerBit) != 0
		}
	}
	return false
}

// LoadPersistentContainers restores every saved durable container's contents
// AND re-applies the persistence ItemBit onto the freshly-parsed room item —
// the bit itself doesn't survive a .SCR/GM-script reload otherwise, only
// MongoDB remembers which instances were flagged persistent. Must be called
// explicitly by main() AFTER LoadGMScripts, not from NewGameEngine: GM-uploaded
// scripts (loaded separately, after the engine constructor returns) can define
// or overwrite rooms built from disk scripts, e.g. a player-home room reusing a
// room number that also exists on disk. Running this before GM scripts load
// would resolve e.rooms against the room that's about to be replaced, so the
// item ref lookup below finds nothing and the restore is silently skipped.
func (e *GameEngine) LoadPersistentContainers() {
	if e.db == nil {
		return
	}
	ctx := context.Background()
	cursor, err := e.db.Collection(persistentContainersCollection).Find(ctx, bson.M{})
	if err != nil {
		return
	}
	var docs []persistentContainerDoc
	if err := cursor.All(ctx, &docs); err != nil {
		return
	}
	loaded := 0
	for _, doc := range docs {
		room := e.rooms[doc.RoomNumber]
		if room == nil {
			continue // room no longer exists in the current scripts
		}
		found := false
		for i := range room.Items {
			if room.Items[i].Ref == doc.ItemRef {
				room.Items[i].ItemBits |= 1 << persistentContainerBit
				found = true
				break
			}
		}
		if !found {
			continue // the chest itself no longer exists at that room/ref
		}
		if len(doc.Contents) > 0 {
			e.roomContainerContents[containerKey(doc.RoomNumber, doc.ItemRef)] = doc.Contents
		}
		loaded++
	}
	if loaded > 0 {
		log.Printf("Loaded %d persistent container(s)", loaded)
	}
}

// savePersistentContainer upserts a durable container's current contents (and
// implicitly its continued existence as a persistent container) to MongoDB.
// Called synchronously (not "go"'d) so the write is durable before the
// triggering command's response is sent — a background write has no guarantee
// of finishing before the process exits (e.g. a redeploy moments later),
// which would silently discard exactly the durability this exists to provide.
func (e *GameEngine) savePersistentContainer(roomNum, ref int, contents []InventoryItem) {
	if e.db == nil {
		return
	}
	ctx := context.Background()
	id := containerKey(roomNum, ref)
	doc := persistentContainerDoc{ID: id, RoomNumber: roomNum, ItemRef: ref, Contents: contents}
	_, err := e.db.Collection(persistentContainersCollection).ReplaceOne(ctx,
		bson.M{"_id": id}, doc, options.Replace().SetUpsert(true))
	if err != nil {
		log.Printf("Failed to save persistent container %s: %v", id, err)
	}
}

// deletePersistentContainer removes a durable container's saved record —
// used both when ITEMBIT19 is explicitly cleared on the instance (no longer
// persistent) and when the container itself is picked up and carried away
// (its contents then travel with the player's inventory item, which already
// persists via the normal player-save path). Synchronous for the same reason
// as savePersistentContainer above.
func (e *GameEngine) deletePersistentContainer(roomNum, ref int) {
	if e.db == nil {
		return
	}
	ctx := context.Background()
	id := containerKey(roomNum, ref)
	if _, err := e.db.Collection(persistentContainersCollection).DeleteOne(ctx, bson.M{"_id": id}); err != nil {
		log.Printf("Failed to delete persistent container %s: %v", id, err)
	}
}
