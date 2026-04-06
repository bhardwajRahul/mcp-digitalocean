## DigitalOcean Volume Tools

This directory provides tools for managing DigitalOcean block storage volumes, volume snapshots, and volume actions via the MCP Server. All operations are exposed as tools with argument-based input, and list endpoints support pagination where applicable.

---

## Supported Tools

### Volume Tools

- **volume-create**  
Create a new block storage volume.  
**Arguments:**  
  - `Name` (string, required): The name of the volume  
  - `SizeGigaBytes` (number, required): The size of the volume in GB  
  - `Region` (string, required): Region slug where the volume will be created  
  - `Description` (string, optional): Human-readable description of the volume  
  - `SnapshotID` (string, optional): Snapshot ID to create the volume from  
  - `FilesystemType` (string, optional): Filesystem type such as `ext4` or `xfs`  
  - `FilesystemLabel` (string, optional): Filesystem label for the volume  
  - `Tags` (array, optional): Tags to apply to the volume
- **volume-list**  
List block storage volumes with optional filters. Supports pagination.  
**Arguments:**  
  - `Name` (string, optional): Name filter  
  - `Region` (string, optional): Region filter  
  - `Page` (number, default: 1): Page number  
  - `PerPage` (number, default: 50): Volumes per page
- **volume-get**  
Get a block storage volume by ID.  
**Arguments:**  
  - `ID` (string, required): The ID of the volume
- **volume-delete**  
Delete a block storage volume by ID.  
**Arguments:**  
  - `ID` (string, required): The ID of the volume to delete

---

### Volume Snapshot Tools

- **volume-snapshot-create**  
Create a new snapshot from a volume.  
**Arguments:**  
  - `VolumeID` (string, required): The ID of the source volume  
  - `Name` (string, required): Snapshot name  
  - `Tags` (array, optional): Tags to apply to the snapshot
- **volume-snapshot-list**  
List snapshots for a volume. Supports pagination.  
**Arguments:**  
  - `VolumeID` (string, required): The ID of the volume  
  - `Page` (number, default: 1): Page number  
  - `PerPage` (number, default: 50): Snapshots per page
- **volume-snapshot-get**  
Get a snapshot by ID.  
**Arguments:**  
  - `ID` (string, required): The ID of the snapshot
- **volume-snapshot-delete**  
Delete a snapshot by ID.  
**Arguments:**  
  - `ID` (string, required): The ID of the snapshot to delete

---

### Volume Action Tools

- **volume-attach**  
Attach a volume to a droplet.  
**Arguments:**  
  - `VolumeID` (string, required): The ID of the volume to attach  
  - `DropletID` (number, required): The ID of the target droplet
- **volume-detach**  
Detach a volume from a droplet.  
**Arguments:**  
  - `VolumeID` (string, required): The ID of the volume to detach  
  - `DropletID` (number, required): The ID of the droplet currently using the volume
- **volume-action-get**  
Get a volume action by ID.  
**Arguments:**  
  - `VolumeID` (string, required): The ID of the volume  
  - `ActionID` (number, required): The ID of the action
- **volume-action-list**  
List actions for a specific volume. Supports pagination.  
**Arguments:**  
  - `VolumeID` (string, required): The ID of the volume  
  - `Page` (number, default: 1): Page number  
  - `PerPage` (number, default: 50): Actions per page
- **volume-resize**  
Resize a volume.  
**Arguments:**  
  - `VolumeID` (string, required): The ID of the volume to resize  
  - `SizeGigaBytes` (number, required): New size in GB  
  - `Region` (string, required): Region slug where the volume exists

---

## Notes

- All tools use argument-based input; do not use resource URIs.
- Pagination is supported on `volume-list`, `volume-snapshot-list`, and `volume-action-list` through `Page` and `PerPage`.
- The implementation caps `PerPage` at `200` for `volume-list`, `volume-snapshot-list`, and `volume-action-list`.
- Numeric MCP arguments are handled as numbers, and in tests they are commonly represented as `float64`.
- `volume-list` returns a filtered summary view of each volume, while `volume-get` returns the full volume object.
- Attach and detach operations return a DigitalOcean action object; use `volume-action-get` or `volume-action-list` to track completion.
- Snapshot tools operate on volume snapshots only; they are separate from Droplet snapshot/image tools.

