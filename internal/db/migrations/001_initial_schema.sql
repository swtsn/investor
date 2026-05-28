CREATE TABLE buckets (
  id         INTEGER PRIMARY KEY,
  name       TEXT NOT NULL UNIQUE,
  type       TEXT NOT NULL CHECK(type IN ('flat','diversified')),
  target_pct TEXT NOT NULL
);

CREATE TABLE allocations (
  id         INTEGER PRIMARY KEY,
  bucket_id  INTEGER NOT NULL REFERENCES buckets(id),
  name       TEXT NOT NULL,
  target_pct TEXT NOT NULL,
  UNIQUE(bucket_id, name)
);

CREATE TABLE budget_events (
  id           INTEGER PRIMARY KEY,
  total_amount TEXT NOT NULL,
  date         TEXT NOT NULL
);

CREATE TABLE contributions (
  id              INTEGER PRIMARY KEY,
  bucket_id       INTEGER NOT NULL REFERENCES buckets(id),
  amount          TEXT NOT NULL,
  origination     TEXT NOT NULL CHECK(origination IN ('budget','reinvestment','slush')),
  budget_event_id INTEGER REFERENCES budget_events(id),
  date            TEXT NOT NULL
);

CREATE TABLE deployments (
  id              INTEGER PRIMARY KEY,
  bucket_id       INTEGER NOT NULL REFERENCES buckets(id),
  allocation_id   INTEGER REFERENCES allocations(id),
  symbol          TEXT,
  shares          TEXT,
  price_per_share TEXT,
  amount          TEXT NOT NULL,
  date            TEXT NOT NULL
);

CREATE TABLE deployment_sources (
  deployment_id   INTEGER NOT NULL REFERENCES deployments(id),
  contribution_id INTEGER NOT NULL REFERENCES contributions(id),
  amount          TEXT NOT NULL,
  PRIMARY KEY (deployment_id, contribution_id)
);
