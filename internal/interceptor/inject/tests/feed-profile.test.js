const assert = require("node:assert/strict");
const fs = require("node:fs");
const path = require("node:path");
const test = require("node:test");
const vm = require("node:vm");

function loadFeedProfileHandler(scriptName) {
  const sourcePath = path.join(__dirname, "..", "src", scriptName);
  const source = fs.readFileSync(sourcePath, "utf8");
  const start = source.indexOf("WXU.onDOMContentLoaded(function () {");
  assert.notEqual(start, -1, "feed profile setup was not found");

  const setFeeds = [];
  let profileHandler;
  const context = {
    setTimeout() {
      return 1;
    },
    clearTimeout() {},
    console,
    WXU: {
      error() {},
      onDOMContentLoaded(handler) {
        handler();
      },
      onFetchFeedProfile(handler) {
        profileHandler = handler;
      },
      onPCFlowLoaded() {},
      onGotoNextFeed() {},
      onGotoPrevFeed() {},
      onHomeFeedChanged() {},
      set_cur_video() {},
      set_feed(feed) {
        setFeeds.push(feed);
      },
      emit() {},
    },
    WXE: { Events: { Feed: "Feed" } },
  };

  vm.createContext(context);
  vm.runInContext(source.slice(start), context);
  assert.equal(typeof profileHandler, "function");
  return { profileHandler, setFeeds };
}

for (const scriptName of ["feed.js", "home.js"]) {
  test(`${scriptName} refreshes the current feed for every detail response`, () => {
    const { profileHandler, setFeeds } = loadFeedProfileHandler(scriptName);
    const first = { id: "feed-1", title: "first" };
    const second = { id: "feed-2", title: "second" };

    profileHandler(first);
    profileHandler(second);

    assert.deepEqual(setFeeds, [first, second]);
  });
}
