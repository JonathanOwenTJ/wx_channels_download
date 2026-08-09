const assert = require("node:assert/strict");
const fs = require("node:fs");
const path = require("node:path");
const test = require("node:test");
const vm = require("node:vm");

function loadFilenameFunctions() {
  const sourcePath = path.join(__dirname, "..", "src", "utils.js");
  const source = fs.readFileSync(sourcePath, "utf8");
  const buildStart = source.indexOf("  function build_filename");
  const helperStart = source.indexOf("  function truncate_utf8_by_bytes");
  const end = source.indexOf("  function remove_zero", buildStart);
  assert.notEqual(buildStart, -1, "build_filename source was not found");
  assert.notEqual(end, -1, "build_filename end marker was not found");

  const context = {
    TextEncoder,
    window: {},
  };
  vm.createContext(context);
  vm.runInContext(source.slice(helperStart === -1 ? buildStart : helperStart, end), context);
  return context.build_filename;
}

const buildFilename = loadFilenameFunctions();
const template = "{{author_folder}}/{{filename}}_{{spec}}";

function profile(contact) {
  return {
    id: "feed_1",
    title: "视频标题",
    createtime: 123,
    contact,
    spec: [{ fileFormat: "1080p" }],
  };
}

test("uses the trimmed nickname as one author directory", () => {
  assert.equal(
    buildFilename(profile({ nickname: " 阿俊说商业 ", id: "wxid_123" }), "1080p", template),
    "阿俊说商业/视频标题_1080p",
  );
});

test("falls back to the video channel id", () => {
  assert.equal(
    buildFilename(profile({ nickname: "", id: " wxid_123 " }), "1080p", template),
    "wxid_123/视频标题_1080p",
  );
});

test("falls back to the fixed unknown directory", () => {
  assert.equal(
    buildFilename(profile({ nickname: "", id: "" }), "1080p", template),
    "未知视频号/视频标题_1080p",
  );
});

test("removes separators and Windows-invalid characters from nickname", () => {
  assert.equal(
    buildFilename(profile({ nickname: `博主/栏目\\精选:*?`, id: "wxid_123" }), "1080p", template),
    "博主栏目精选/视频标题_1080p",
  );
});

test("falls back when the cleaned nickname is unusable", () => {
  assert.equal(
    buildFilename(profile({ nickname: "..", id: "wxid_123" }), "1080p", template),
    "wxid_123/视频标题_1080p",
  );
});

test("makes Windows reserved directory names safe", () => {
  assert.equal(
    buildFilename(profile({ nickname: "CON", id: "wxid_123" }), "1080p", template),
    "CON_/视频标题_1080p",
  );
});

test("removes path separators from the video title", () => {
  const item = profile({ nickname: "案痕推理薄", id: "wxid_123" });
  item.title = String.raw`标题①/②\③:*?`;
  assert.equal(
    buildFilename(item, "1080p", template),
    "案痕推理薄/标题①②③_1080p",
  );
});
