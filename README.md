# Changing the identity_pubkey of an lnd node

Tested on lnd v0.16.0-beta with the following patch applied: https://github.com/lightningnetwork/lnd/pull/7938/files

```diff
--- a/peer/brontide.go
+++ b/peer/brontide.go
@@ -672,7 +672,7 @@ func (p *Brontide) Start() error {
 	//
 	// TODO(wilmer): Remove this once we're able to query for node
 	// announcements through their timestamps.
-	p.maybeSendNodeAnn(activeChans)
+	go p.maybeSendNodeAnn(activeChans)
 
 	return nil
 }
```

lnd2: Node to change the identity of

lnd1: Peer of lnd2

## Step 1: Write down the old identity public key

```console
admin@lnd2:~$ lncli getinfo
```

Write down the old identity_pubkey

## Step 2: Stop lnd2

```console
admin@lnd2:~$ sudo systemctl stop lnd
```

## Step 3: Edit the source code of lnd for lnd2

```console
admin@lnd2:~/lnd$ git apply ../lnd-change-node-id.patch
```

```diff
--- a/keychain/derivation.go	
+++ b/keychain/derivation.go
@@ -100,7 +100,7 @@
 	// "identity" within the network. Peers will need our latest node key
 	// in order to establish a transport session with us on the Lightning
 	// p2p level (BOLT-0008).
-	KeyFamilyNodeKey KeyFamily = 6
+	KeyFamilyNodeKey KeyFamily = 16
 
 	// KeyFamilyBaseEncryption is the family of keys that will be used to
 	// derive keys that we use to encrypt and decrypt any general blob data
```

This value has to change in order for the own node to get a new identity public key. It does not matter what you set it to, it just has to be a different value.

Also edit the addinvoice code so that the channel lnd2 has to lnd1 will be added to invoices as a routing hint, since the channel will not be reannounced to the network with the new node id of lnd2.

```console
admin@lnd2:~/lnd$ git apply ../lnd-public-channel-routing-hints.patch
```

```diff
--- a/lnrpc/invoicesrpc/addinvoice.go
+++ b/lnrpc/invoicesrpc/addinvoice.go
@@ -484,12 +484,6 @@
 func chanCanBeHopHint(channel *HopHintInfo, cfg *SelectHopHintsCfg) (
 	*channeldb.ChannelEdgePolicy, bool) {
 
-	// Since we're only interested in our private channels, we'll skip
-	// public ones.
-	if channel.IsPublic {
-		return nil, false
-	}
-
 	// Make sure the channel is active.
 	if !channel.IsActive {
 		log.Debugf("Skipping channel %v due to not "+
@@ -694,10 +688,7 @@ func getPotentialHints(cfg *SelectHopHintsCfg) ([]*channeldb.OpenChannel,
 
 	privateChannels := make([]*channeldb.OpenChannel, 0, len(openChannels))
 	for _, oc := range openChannels {
-		isPublic := oc.ChannelFlags&lnwire.FFAnnounceChannel != 0
-		if !isPublic {
-			privateChannels = append(privateChannels, oc)
-		}
+		privateChannels = append(privateChannels, oc)
 	}
 
 	// Sort the channels in descending remote balance.
```

## Step 4: Build lnd

```console
admin@lnd2:~/lnd$ make install
```

## Step 5: Start lnd2

```console
admin@lnd2:~$ sudo systemctl start lnd
```

## Step 6: Write down the new identity_pubkey

```console
admin@lnd2:~$ lncli getinfo
```

Write down the new identity_pubkey

## Step 7: Stop lnd2

```console
admin@lnd2:~$ sudo systemctl stop lnd
```

## Step 8: Make a backup of the channel.db on lnd2

```console
admin@lnd2:~$ cp .lnd/data/graph/mainnet/channel.db .lnd/data/graph/mainnet/channel.db-orig
```

## Step 9: Replace all occurrences of the old node id with the new node id in the channel database of lnd2

Note: To build bbolt-replace run `go build` in the bbolt-replace directory.

```console
admin@lnd2:~$ bbolt-replace/bbolt-replace .lnd/data/graph/mainnet/channel.db <old identity_pubkey> <new identity_pubkey>
```

## Step 10: Stop lnd1

```console
admin@lnd1:~$ sudo systemctl stop lnd
```

## Step 11: Make a backup of the channel.db on lnd1

```console
admin@lnd1:~$ cp .lnd/data/graph/mainnet/channel.db .lnd/data/graph/mainnet/channel.db-orig
```

## Step 12: Replace all occurrences of the old node id with the new node id in the channel database of lnd1

Note: To build bbolt-replace run `go build` in the bbolt-replace directory.

```console
admin@lnd1:~$ bbolt-replace/bbolt-replace .lnd/data/graph/mainnet/channel.db <old identity_pubkey> <new identity_pubkey>
```

## Step 13: Start lnd1 & lnd2

```console
admin@lnd1:~$ sudo systemctl start lnd
```

```console
admin@lnd2:~$ sudo systemctl start lnd
```
