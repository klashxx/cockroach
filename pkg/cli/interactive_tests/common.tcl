# We are sending vt100 terminal sequences below, so inform readline
# accordingly.
set env(TERM) vt100

# Keep the history in a test location, so as to not override the
# developer's own history file.
set histfile ".cockroachdb_history_test"
set ::env(COCKROACH_SQL_CLI_HISTORY) $histfile
system "rm -f $histfile"

# Everything in this test should be fast. Don't be tolerant for long
# waits.
set timeout 30

# When run via Docker the enclosing terminal has 0 columns and 0 rows,
# and this confuses readline. Ensure sane defaults here.
set stty_init "cols 80 rows 25"

# Convenience wrapper function, which ensures that all expects are
# mandatory (i.e. with a mandatory fail if the expected output doesn't
# show up fast).
proc eexpect {text} {
    expect {
	$text {}
	timeout {exit 1}
    }
}

# Convenience function that sends Ctrl+C to the monitored process.
proc interrupt {} {
    send "\003"
    sleep 0.4
}

# Tests that wish to concatenate a marker with the first line of
# output need to exclude messages about the env. var
# PROPOSER_EVALUATED_KV as this may be emitted before the logging
# flags are handled.
set silence_prop_eval_kv "2>&1 | grep -v 'running with experimental support for proposer-evaluated KV'"

# Convenience functions to start/shutdown the server.
# Preserves the invariant that the server's PID is saved
# in `server_pid`.
proc start_server {argv} {
    system "$argv start & echo \$! > server_pid"
    sleep 1
}
proc stop_server {argv} {
    system "set -e; if kill -CONT `cat server_pid`; then $argv quit || true & sleep 1; kill -9 `cat server_pid` || true; else $argv quit || true; fi"
}
