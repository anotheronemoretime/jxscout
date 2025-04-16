import { parse, Program } from "acorn";
import { is, traverse } from "estree-toolkit";
import Sval from "sval";

export interface ChunkLoaderOptions {
  basePath: string;
  fileExtension: string;
  bruteforceLimit: number;
}

export class ChunkLoader {
  private options: ChunkLoaderOptions;
  private loadedChunks: Set<string> = new Set();

  constructor(options: ChunkLoaderOptions) {
    this.options = options;
  }

  public async loadChunks(url: string): Promise<string[]> {
    try {
      const response = await fetch(url);
      if (!response.ok) {
        throw new Error(`HTTP error! status: ${response.status}`);
      }
      const scriptContent = await response.text();
      
      // Try to parse as AST first
      let ast;
      try {
        ast = parse(scriptContent, {
          ecmaVersion: "latest",
          sourceType: "module",
        });
      } catch (err) {
        ast = parse(scriptContent, {
          ecmaVersion: "latest",
          sourceType: "script",
        });
      }

      // Try different chunk discovery methods
      const chunks = await this.discoverChunks(scriptContent, ast);
      return [...chunks];
    } catch (error) {
      console.error('Error loading chunks:', error);
      return [];
    }
  }

  public async discoverChunks(scriptContent: string, ast: Program): Promise<string[]> {
    const chunks = new Set<string>();

    // Next.js manifest detection
    const nextJsManifestFunctionRegex = /self\.__BUILD_MANIFEST\s*=\s*(function\s*\([^\)]*\)?\s*\{[\s\S]*?\}\s*\([^)]*\));?/;
    const nextJsManifestObjectRegex = /self\.__BUILD_MANIFEST\s*=\s*({[\s\S]*})/;
    
    const nextJsManifestFunctionMatch = scriptContent.match(nextJsManifestFunctionRegex);
    const nextJsManifestObjectMatch = scriptContent.match(nextJsManifestObjectRegex);

    if (nextJsManifestFunctionMatch) {
      const manifestChunks = await this.handleNextJsManifestFunction(nextJsManifestFunctionMatch[1]);
      manifestChunks.forEach(chunk => chunks.add(chunk));
    } else if (nextJsManifestObjectMatch) {
      const manifestChunks = await this.handleNextJsManifestObject(nextJsManifestObjectMatch[1]);
      manifestChunks.forEach(chunk => chunks.add(chunk));
    }

    // Webpack chunk detection
    const webpackChunks = this.discoverWebpackChunks(ast);
    webpackChunks.forEach(chunk => chunks.add(chunk));

    // Modern chunk detection
    const modernChunkRegex = /return\s+o\.p\s*\+\s*""\s*\+\s*\{([\s\S]*?)\}/;
    const modernChunkMatch = scriptContent.match(modernChunkRegex);
    if (modernChunkMatch) {
      const modernChunks = this.handleModernChunks(modernChunkMatch[1]);
      modernChunks.forEach(chunk => chunks.add(chunk));
    }

    return [...chunks];
  }

  private async handleNextJsManifestFunction(manifestFunctionCall: string): Promise<string[]> {
    const sval = new Sval({
      ecmaVer: "latest",
      sourceType: "script",
    });

    sval.run(`exports.res = (function() { return ${manifestFunctionCall}; })()`);
    const buildManifest = sval.exports.res;
    
    const chunks: string[] = [];
    for (const key in buildManifest) {
      if (Array.isArray(buildManifest[key])) {
        chunks.push(...buildManifest[key]);
      }
    }
    return chunks;
  }

  private async handleNextJsManifestObject(manifestObjectString: string): Promise<string[]> {
    const sval = new Sval({
      ecmaVer: "latest",
      sourceType: "script",
    });

    sval.run(`exports.res = (${manifestObjectString})`);
    const buildManifest = sval.exports.res;
    
    const chunks: string[] = [];
    for (const key in buildManifest) {
      if (Array.isArray(buildManifest[key])) {
        chunks.push(...buildManifest[key]);
      }
    }
    return chunks;
  }

  private handleModernChunks(chunkMapString: string): string[] {
    const chunkMap = this.parseChunkMap(chunkMapString);
    return Object.values(chunkMap).map(value => `${value}.modern.js`);
  }

  private discoverWebpackChunks(ast: Program): string[] {
    const chunks = new Set<string>();
    
    traverse(ast, {
      CallExpression(path) {
        if (is.identifier(path.node.callee) && 
            path.node.callee.name === 'require' && 
            path.node.arguments.length === 1) {
          const arg = path.node.arguments[0];
          if (is.literal(arg) && typeof arg.value === 'string') {
            chunks.add(arg.value);
          }
        }
      },
      ImportDeclaration(path) {
        if (is.literal(path.node.source)) {
          chunks.add(path.node.source.value);
        }
      }
    });

    return [...chunks];
  }

  private parseChunkMap(chunkMapString: string): Record<string, string> {
    const trimmed = chunkMapString.replace(/[{}]/g, '').trim();
    return trimmed.split(/\s*,\s*/).reduce((acc, pair) => {
      const [key, value] = pair.split(/\s*:\s*/);
      acc[key] = value.replace(/"/g, '');
      return acc;
    }, {} as Record<string, string>);
  }
} 