import {editor} from 'monaco-editor';
import IEditorOptions = editor.IEditorOptions;

export const DEFAULT_EDITOR_OPTIONS: IEditorOptions = {
  automaticLayout: true,
  lineNumbers: 'on',  // Включение номеров строк
  // -------------------
  // folding: false,
  // Undocumented see https://github.com/Microsoft/vscode/issues/30795#issuecomment-410998882
  lineDecorationsWidth: 4,
  lineNumbersMinChars: 2,
  // -------------------

  contextmenu: false,

  fontSize: 13,
  lineHeight: 1.6,
  // hover: {enabled: false},
  // renderWhitespace: 'none',
  wordWrap: 'on',
  scrollbar: {
    vertical: "hidden",
    horizontal: "hidden",
    // verticalScrollbarSize: 0,
    alwaysConsumeMouseWheel: false,
    useShadows: false,
  },


  overviewRulerBorder: false,
  overviewRulerLanes: 0,
  fontFamily: 'JetBrains Mono',
  minimap: {
    enabled: false,
  },
  renderLineHighlightOnlyWhenFocus: false,
  suggestOnTriggerCharacters: false,
  quickSuggestions: false,
  parameterHints: {
    enabled: false
  },
  acceptSuggestionOnEnter: "off",
  tabCompletion: "off",
  folding: false,
  foldingHighlight: false,
  selectionHighlight: false,
  selectOnLineNumbers: false,
  scrollBeyondLastLine: false,
  hideCursorInOverviewRuler: true,
  renderLineHighlight: "none",
  cursorWidth: 2,

  matchBrackets: "near",

  guides: {
    indentation: false,
  },

  padding: {
    top: 15,
    bottom: 15,
  },
}
