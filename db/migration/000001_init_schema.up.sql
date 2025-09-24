CREATE TABLE "accounts" (
  "id" bigserial PRIMARY KEY,
  "owner" varchar NOT NULL,
  "created_at" timestamptz NOT NULL DEFAULT (now())
);

CREATE TABLE "waterMeterStatement" (
  "id" bigserial PRIMARY KEY,
  "time" text NOT NULL,
  "consumptionDay" real NOT NULL,
  "consumtionHour" real NOT NULL UNIQUE, 
  "created_at" timestamptz NOT NULL DEFAULT (now())
);

CREATE TABLE "transfers" (
  "id" bigserial PRIMARY KEY,
  "time" text NOT NULL,
  "innings" real NOT NULL,
  "return" real NOT NULL,
  "consumtionHour" real NOT NULL,
  "T1" int NOT NULL,
  "T2" int NOT NULL,
  "amount" bigint NOT NULL,
  "created_at" timestamptz NOT NULL DEFAULT (now())
);

CREATE INDEX ON "accounts" ("owner");

CREATE INDEX ON "waterMeterStatement" ("id");

CREATE INDEX ON "transfers" ("consumtionHour");

CREATE INDEX ON "transfers" ("T1");

COMMENT ON COLUMN "transfers"."amount" IS 'must be positive';

ALTER TABLE "transfers" ADD FOREIGN KEY ("consumtionHour") REFERENCES "waterMeterStatement" ("consumtionHour");
