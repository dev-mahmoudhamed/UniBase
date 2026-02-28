// ==============================
// MongoDB Full Server Metadata
// ==============================

function getServerMetadata() {
  const metadata = {};

  // -----------------------------
  // 1️⃣ Server Level Information
  // -----------------------------
  metadata.serverStatus = db.serverStatus();
  metadata.buildInfo = db.runCommand({ buildInfo: 1 });
  metadata.hostInfo = db.runCommand({ hostInfo: 1 });
  metadata.serverCmdLineOpts = db.runCommand({ getCmdLineOpts: 1 });

  // Replica Set Info (if exists)
  try {
    metadata.replicaSetStatus = rs.status();
  } catch (e) {
    metadata.replicaSetStatus = "Not a replica set";
  }

  // Sharding Info
  try {
    metadata.shardingInfo = db.adminCommand({ listShards: 1 });
  } catch (e) {
    metadata.shardingInfo = "Sharding not enabled";
  }

  // -----------------------------
  // 2️⃣ Databases Metadata
  // -----------------------------
  const dbs = db.adminCommand({ listDatabases: 1 });
  metadata.databases = [];

  dbs.databases.forEach((database) => {
    const dbName = database.name;
    const databaseInfo = {
      name: dbName,
      sizeOnDisk: database.sizeOnDisk,
      empty: database.empty,
      collections: [],
    };

    const currentDb = db.getSiblingDB(dbName);
    const collections = currentDb.getCollectionNames();

    collections.forEach((collectionName) => {
      const collection = currentDb.getCollection(collectionName);

      let stats = {};
      try {
        stats = collection.stats();
      } catch (e) {
        stats = { error: "Unable to retrieve stats" };
      }

      let indexes = [];
      try {
        indexes = collection.getIndexes();
      } catch (e) {
        indexes = [];
      }

      databaseInfo.collections.push({
        name: collectionName,
        stats: stats,
        indexes: indexes,
      });
    });

    metadata.databases.push(databaseInfo);
  });

  // -----------------------------
  // 3️⃣ Users & Roles (if allowed)
  // -----------------------------
  try {
    metadata.users = db.getSiblingDB("admin").system.users.find().toArray();
  } catch (e) {
    metadata.users = "Not authorized to view users";
  }

  return metadata;
}

// Print as formatted JSON
printjson(getServerMetadata());
