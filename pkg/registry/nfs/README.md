## DigitalOcean NFS (File Share) Tools

This directory provides tools for managing DigitalOcean NFS file shares, NFS snapshots, and file share actions via the MCP server. All operations are exposed as tools with argument-based input, and list endpoints support pagination where applicable.

---

## Supported Tools

### File share tools

- **nfs-file-share-create**  
Create a new NFS file share.  
**Arguments:**  
  - `Name` (string, required): Name of the file share  
  - `SizeGibibytes` (number, required): Size of the file share in GiB (minimum 50)  
  - `Region` (string, required): Region slug where the file share will be created  
  - `VpcIds` (array, optional): VPC UUIDs to associate with the file share  
  - `PerformanceTier` (string, optional): Performance tier for the file share
- **nfs-file-share-list**  
List NFS file shares with optional region filtering. Supports pagination.  
**Arguments:**  
  - `Region` (string, optional): Region filter  
  - `Page` (number, default: 1): Page number  
  - `PerPage` (number, default: 20): File shares per page
- **nfs-file-share-get**  
Get a file share by ID.  
**Arguments:**  
  - `ID` (string, required): ID of the file share
- **nfs-file-share-delete**  
Delete a file share by ID.  
**Arguments:**  
  - `ID` (string, required): ID of the file share to delete

---

### NFS snapshot tools

- **nfs-snapshot-list**  
List NFS snapshots with optional filters. Supports pagination.  
**Arguments:**  
  - `Region` (string, optional): Region filter  
  - `ShareID` (string, optional): Only list snapshots for this file share ID  
  - `Page` (number, default: 1): Page number  
  - `PerPage` (number, default: 20): Snapshots per page
- **nfs-snapshot-get**  
Get an NFS snapshot by ID.  
**Arguments:**  
  - `ID` (string, required): ID of the snapshot
- **nfs-snapshot-delete**  
Delete an NFS snapshot by ID.  
**Arguments:**  
  - `ID` (string, required): ID of the snapshot to delete

---

### File share action tools

- **nfs-resize**  
Resize a file share.  
**Arguments:**  
  - `ShareID` (string, required): ID of the file share to resize  
  - `SizeGibibytes` (number, required): New size in GiB (minimum 50)
- **nfs-snapshot**  
Create a snapshot of a file share (asynchronous action).  
**Arguments:**  
  - `ShareID` (string, required): ID of the file share to snapshot  
  - `SnapshotName` (string, required): Name for the new snapshot
- **nfs-attach**  
Attach a file share to a VPC.  
**Arguments:**  
  - `ShareID` (string, required): ID of the file share  
  - `VpcID` (string, required): ID of the VPC to attach to
- **nfs-detach**  
Detach a file share from a VPC.  
**Arguments:**  
  - `ShareID` (string, required): ID of the file share  
  - `VpcID` (string, required): ID of the VPC to detach from
- **nfs-reassign**  
Reassign a file share from one VPC to another.  
**Arguments:**  
  - `ShareID` (string, required): ID of the file share  
  - `OldVpcID` (string, required): Current VPC ID  
  - `NewVpcID` (string, required): Target VPC ID
- **nfs-switch-performance-tier**  
Change the performance tier of a file share.  
**Arguments:**  
  - `ShareID` (string, required): ID of the file share  
  - `PerformanceTier` (string, required): Target performance tier

---

## Notes

- All tools use argument-based input; do not use resource URIs.
- Pagination is supported on `nfs-file-share-list` and `nfs-snapshot-list` through `Page` and `PerPage`.
- The implementation caps `PerPage` at `50` for both list tools.
- Numeric MCP arguments are handled as numbers, and in tests they are commonly represented as `float64`.
- `nfs-file-share-list` returns a filtered summary of each file share; `nfs-file-share-get` returns the full file share object.
- `nfs-snapshot-list` returns a summary of each snapshot; `nfs-snapshot-get` returns the full snapshot object.
- Resize, snapshot, attach, detach, reassign, and performance-tier tools return a DigitalOcean NFS action object (JSON) for tracking asynchronous work.
- The action tool `**nfs-snapshot`** creates a snapshot job for a file share; `**nfs-snapshot-list**`, `**nfs-snapshot-get**`, and `**nfs-snapshot-delete**` operate on snapshot resources by ID.

