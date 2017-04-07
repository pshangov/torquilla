Torquilla
=========

Lightweight unopinionated database migrations.

Usage
-----

Torquilla is a tool that combines multiple sql scripts within a revision range into a single sql file. This file can then be used to execute a database migration between the two revisions using any database shell or admin tool.

    // generate a migration script for changes between the specified revisions
    $> tq 9ce93f4 7148a6a > migrate.sql

    // generate a migration script for changes between the specified revision and HEAD
    $> tq 9ce93f4 > migrate.sql

Basic concepts
--------------

### Git

Database schema changes are versioned using git. A database version is just a git revision.

### Migrations vs. definitions

A migration script changes the state of the schema or data. Examples of migrations scripts are adding a column or table, or moving or changing data. A definition script describes a logical entity, e.g. a stored procedure or a view. Definition scripts are idempotent, i.e. they can be safely run multiple times and will always produce the same results. Migration scripts **update** existing entities, where as defnition scripts **replace** existing entities.

### Manifest

SQL should be listed and ordered in a manifest file, and will be output according to their order there.

### Rollbacks

Torquilla currently does not support rollbacks. A rollback should be manually implemented as a forward migration reverting the problematic changes.

Configuration
-------------

The root of the database repo should contain a `torquilla.[toml|json|yaml]` configuration file. The following options are supported:

* `migrations` list of paths where migration scripts are located
* `definitions` list of paths where definition scripts are located
* `extensions` if provided, only files with the specified extensions will be included
* `template` template for output, variables available are `Script` and `Sha`

Example:

    migrations   = [ "migrations" ]
    definitions  = [ "procedures" ]
    extensions   = [ ".sql" ]
    template    = """
      {{.Script}}
      -- Insert new version number
      INSERT INTO version_history (version_sha) VALUES ('{{.Sha}}');"""

File selection and ordering
---------------------------

Only migration scripts that have been newly added in the revision range will be included. Both new and modified definition scripts will be included.

Scripts are orderd by commit timestamp. The ordering of scripts within a single commit is currently not guaranteed.
