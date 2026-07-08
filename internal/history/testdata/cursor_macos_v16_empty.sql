CREATE TABLE cursorDiskKV (key TEXT UNIQUE ON CONFLICT REPLACE, value BLOB);
CREATE TABLE composerHeaders (composerId TEXT PRIMARY KEY, workspaceId TEXT, createdAt INTEGER, lastUpdatedAt INTEGER, isArchived INTEGER, isSubagent INTEGER, recency INTEGER, checkpointAt INTEGER, value TEXT);

INSERT INTO cursorDiskKV (key, value) VALUES (
  'composerData:sample-empty',
  '{"_v":16,"composerId":"sample-empty","richText":"","hasLoaded":true,"text":"","fullConversationHeadersOnly":[],"conversationMap":{},"status":"none","context":{"composers":[],"selectedCommits":[]},"createdAt":1783444765134,"isAgentic":true}'
);
