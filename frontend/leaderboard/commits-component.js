// CommitsComponent: React component that subscribes to the commits-service
// SSE stream and renders a live list of recent git commits.
// Uses React.createElement only (no JSX, no build step required).
// React and ReactDOM are loaded as UMD bundles before this module is imported.

const { useState, useEffect } = React;

const POLL_INTERVAL_MS = 30000;

// CommitsComponent renders the commit feed column contents.
// Props: { commitsServiceURL: string }
function CommitsComponent({ commitsServiceURL }) {
  // null = loading (no data yet), [] = loaded but empty, [...] = has commits
  const [commits, setCommits] = useState(null);

  useEffect(() => {
    let cleanup = null;

    if (typeof EventSource !== 'undefined') {
      cleanup = startSSE(commitsServiceURL, setCommits);
    } else {
      cleanup = startPolling(commitsServiceURL, setCommits);
    }

    return cleanup;
  }, [commitsServiceURL]);

  if (commits === null) {
    // Still waiting for first event — render nothing to avoid flash.
    return null;
  }

  if (commits.length === 0) {
    return React.createElement(
      'p',
      { id: 'commit-status' },
      'No commits yet — make your first commit to see it here!'
    );
  }

  return React.createElement(
    'ul',
    { id: 'commits' },
    commits.map(function (c) {
      return React.createElement(
        'li',
        { key: c.hash + c.date },
        React.createElement('span', { className: 'chash' }, c.hash),
        React.createElement('span', { className: 'cauthor' }, c.author),
        React.createElement('span', { className: 'cdate' }, c.date),
        React.createElement('span', { className: 'cmsg' }, c.message)
      );
    })
  );
}

// startSSE opens an EventSource connection to the commits stream.
// On each "commits" event, updates state. On permanent error, falls back to
// polling. Returns a cleanup function.
function startSSE(baseURL, setCommits) {
  const source = new EventSource(baseURL + '/commits/stream');
  let pollingCleanup = null;

  source.addEventListener('commits', function (e) {
    try {
      const data = JSON.parse(e.data);
      setCommits(data.commits || []);
    } catch (_) {}
  });

  source.onerror = function () {
    if (source.readyState === EventSource.CLOSED) {
      // Permanent failure — fall back to polling.
      source.close();
      pollingCleanup = startPolling(baseURL, setCommits);
    }
    // CONNECTING state: browser will auto-reconnect; do nothing.
  };

  return function () {
    source.close();
    if (pollingCleanup) pollingCleanup();
  };
}

// startPolling fetches the commit feed on a fixed interval.
// Returns a cleanup function.
function startPolling(baseURL, setCommits) {
  function fetchCommits() {
    fetch(baseURL + '/commits')
      .then(function (resp) {
        if (!resp.ok) throw new Error('commits fetch failed: ' + resp.status);
        return resp.json();
      })
      .then(function (data) {
        setCommits(data.commits || []);
      })
      .catch(function () {
        // Leave last known state intact on transient failure.
      });
  }

  fetchCommits();
  const id = setInterval(fetchCommits, POLL_INTERVAL_MS);
  return function () { clearInterval(id); };
}

export { CommitsComponent };
