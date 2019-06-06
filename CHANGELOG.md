## v3.0 (Release date TBD)

Dragonboat v3.0 is a major release with breaking changes. Please make sure to carefully read the CHANGELOG below before upgrading.

### New features

- Added on disk state machine (statemachine.IOnDiskStateMachine) support. The on disk state machine is close to the concept described in the section 5.2 of Diego Ongaro's Raft thesis. 
- Added new API for requesting a  snapshot to be created or exported.
- Added the ability to use exported snapshot to repair permanently damaged cluster that no longer has its majority quorum.
- Added new API for cleaning up data and release disk spaces after a node is removed from its Raft cluster.
- Added the ability to limit peak memory usage when disk or network is slow.
- Added Go module support. Go 1.12 is required.

### Improvements

- Further improved self checking on configurations.
- Added snapshot binary format version 2 with block base checksum.
- Synchronous variants have been provided for all asynchronous user request APIs in NodeHost.

### Breaking changes

- The Drummer package has been made invisible from user applications.
- The statemachine.IStateMachine interface has been upgraded to reflect the fact that not all state machine data is stored in memory ([#46](https://github.com/lni/dragonboat/issues/46)).

## v2.1 (Feb 2019)

### New features

- Added support to store Raft Logs in LevelDB.

## v2.0 (Jan 2019)

Initial open source release. 