import fs from "fs";
import path from "path";
import { expect, test } from "vitest";
import { ChunkLoader } from "../discoverer/chunk-loader.js";
import { parse } from "acorn";

const readFile = (path: string): string => {
  return fs.readFileSync(path, "utf-8");
};

interface ChunkLoaderTestCase {
  jsFileName: string;
  expectedChunks: string[];
  description: string;
}

const testCases: ChunkLoaderTestCase[] = [
  {
    jsFileName: "nextjs-manifest-function.js",
    expectedChunks: [
      "static/chunks/0.js",
      "static/chunks/1.js",
      "static/chunks/2.js",
      "static/chunks/3.js",
      "static/chunks/4.js",
    ],
    description: "should discover chunks from Next.js manifest function",
  },
  {
    jsFileName: "nextjs-manifest-object.js",
    expectedChunks: [
      "static/chunks/0.js",
      "static/chunks/1.js",
      "static/chunks/2.js",
      "static/chunks/3.js",
      "static/chunks/4.js",
    ],
    description: "should discover chunks from Next.js manifest object",
  },
  {
    jsFileName: "modern-chunks.js",
    expectedChunks: [
      "chunk1.modern.js",
      "chunk2.modern.js",
      "chunk3.modern.js",
    ],
    description: "should discover modern chunks",
  },
  {
    jsFileName: "webpack-chunks.js",
    expectedChunks: [
      "chunk1.js",
      "chunk2.js",
      "chunk3.js",
    ],
    description: "should discover Webpack chunks",
  },
  {
    jsFileName: "mixed-chunks.js",
    expectedChunks: [
      "static/chunks/0.js",
      "static/chunks/1.js",
      "chunk1.modern.js",
      "chunk2.modern.js",
      "webpack-chunk1.js",
      "webpack-chunk2.js",
    ],
    description: "should discover mixed chunks from different sources",
  },
];

testCases.forEach(({ jsFileName, expectedChunks, description }) => {
  test(description, async () => {
    const filePath = path.join(__dirname, "fixtures", jsFileName);
    const code = readFile(filePath);
    
    // Parse the code to get AST
    let ast;
    try {
      ast = parse(code, {
        ecmaVersion: "latest",
        sourceType: "module",
      });
    } catch (err) {
      ast = parse(code, {
        ecmaVersion: "latest",
        sourceType: "script",
      });
    }

    // Initialize chunk loader
    const chunkLoader = new ChunkLoader({
      basePath: "",
      fileExtension: ".js",
      bruteforceLimit: 1000,
    });

    // Discover chunks
    const chunks = await chunkLoader.discoverChunks(code, ast);

    // Sort arrays for comparison
    const sortedExpected = [...expectedChunks].sort();
    const sortedActual = [...chunks].sort();

    expect(sortedActual).toEqual(sortedExpected);
  });
});

// Test error handling
test("should handle invalid code gracefully", async () => {
  const chunkLoader = new ChunkLoader({
    basePath: "",
    fileExtension: ".js",
    bruteforceLimit: 1000,
  });

  const chunks = await chunkLoader.discoverChunks("invalid code", {} as any);
  expect(chunks).toEqual([]);
});

// Test Next.js manifest function error handling
test("should handle invalid Next.js manifest function", async () => {
  const chunkLoader = new ChunkLoader({
    basePath: "",
    fileExtension: ".js",
    bruteforceLimit: 1000,
  });

  const code = "self.__BUILD_MANIFEST = function() { return { invalid: 'object' }; }();";
  const ast = parse(code, { ecmaVersion: "latest", sourceType: "script" });
  
  const chunks = await chunkLoader.discoverChunks(code, ast);
  expect(chunks).toEqual([]);
});

// Test Next.js manifest object error handling
test("should handle invalid Next.js manifest object", async () => {
  const chunkLoader = new ChunkLoader({
    basePath: "",
    fileExtension: ".js",
    bruteforceLimit: 1000,
  });

  const code = "self.__BUILD_MANIFEST = { invalid: 'object' };";
  const ast = parse(code, { ecmaVersion: "latest", sourceType: "script" });
  
  const chunks = await chunkLoader.discoverChunks(code, ast);
  expect(chunks).toEqual([]);
});

// Test modern chunks error handling
test("should handle invalid modern chunks format", async () => {
  const chunkLoader = new ChunkLoader({
    basePath: "",
    fileExtension: ".js",
    bruteforceLimit: 1000,
  });

  const code = "function test() { return o.p + '' + { invalid: 'format' }; }";
  const ast = parse(code, { ecmaVersion: "latest", sourceType: "script" });
  
  const chunks = await chunkLoader.discoverChunks(code, ast);
  expect(chunks).toEqual([]);
}); 