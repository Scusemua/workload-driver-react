import {DarkModeContext} from '@app/Providers';
import {Monaco} from '@monaco-editor/react';
import {CodeEditor, CodeEditorControl, Language} from '@patternfly/react-code-editor';
import {Button, Chip, Grid, GridItem, Switch, TextInput} from '@patternfly/react-core';
import {CodeIcon, FileIcon} from '@patternfly/react-icons';
import {editor} from 'monaco-editor/esm/vs/editor/editor.api';
import React from 'react';
import {CodeContext} from './Modals';

export interface CodeEditorComponentProps {
  children?: React.ReactNode;
  showCodeTemplates: boolean;
  height: number;
  language: Language;

  // Do not include the file extension. That is added automatically.
  defaultFilename: string;
}

export const CodeEditorComponent: React.FunctionComponent<CodeEditorComponentProps> = (props: CodeEditorComponentProps) => {
  const {darkMode} = React.useContext(DarkModeContext);
  const {code, setCode} = React.useContext(CodeContext);

  const [isEditorDarkMode, setIsEditorDarkMode] = React.useState(darkMode);
  const [filename, setFilename] = React.useState<string>('');

  // If the default filename specified in the props is empty, then use a different default value.
  const defaultFilename: string = props.defaultFilename.length == 0 ? Date.now().toString() : props.defaultFilename;

  const onEditorDidMount = (editor: editor.IStandaloneCodeEditor, monaco: Monaco) => {
    editor.layout();
    editor.focus();
    monaco.editor.getModels()[0].updateOptions({tabSize: 5});
  };

  const shortcuts = [
    {
      keys: ['Opt', 'F1'],
      description: 'Accessibility helps',
    },
    {
      keys: ['F1'],
      description: 'View all editor shortcuts',
    },
    {
      keys: ['Ctrl', 'Space'],
      description: 'Activate auto complete',
    },
    {
      keys: ['Cmd', 'S'],
      description: 'Save',
    },
  ];
  const shortcutsPopoverProps = {
    bodyContent: (
      <Grid span={6} hasGutter key="grid">
        {shortcuts.map((shortcut, index) => (
          <React.Fragment key={index}>
            <GridItem style={{textAlign: 'right', marginRight: '1em'}}>
              {shortcut.keys
                .map((key) => (
                  <Chip key={key} isReadOnly>
                    {key}
                  </Chip>
                ))
                .reduce((prev, curr) => (
                  <>{[prev, ' + ', curr]}</>
                ))}
            </GridItem>
            <GridItem>{shortcut.description}</GridItem>
          </React.Fragment>
        ))}
      </Grid>
    ),
    'aria-label': 'Shortcuts',
  };

  // Function to check if a given filename is valid (for Windows).
  const isValidFilename = (fname: string) => {
    const rg1 = /^[^\\/:*?"<>|]+$/; // forbidden characters \ / : * ? " < > |
    const rg2 = /^\./; // cannot start with dot (.)
    const rg3 = /^(nul|prn|con|lpt[0-9]|com[0-9])(\.|$)/i; // forbidden file names
    return rg1.test(fname) && !rg2.test(fname) && !rg3.test(fname);
  }

  // Function to check if the filename entered by the user.
  // If the filename is empty, we use a default filename, which is a special case insofar
  // as the 'isValidFilename' function returns false for empty strings.
  const isUserFilenameValid = () => {
    if (!filename || filename.length == 0 || isValidFilename(filename)) {
      return 'success';
    }

    return 'error';
  }

  const fileNameField = (
    <TextInput
      customIcon={<FileIcon/>}
      // If the user hasn't specified a filename, then don't add the file extension automatically.
      // We'll use the placeholder text instead.
      value={filename}
      type="text"
      onChange={(_event, value) => setFilename(value)}
      placeholder={defaultFilename}
      validated={isUserFilenameValid()}
    />
  );

  const darkLightThemeSwitch = (
    <div>
      <Button
        variant="link"
        onClick={(event: React.MouseEvent<HTMLButtonElement, MouseEvent>) => {
          event.stopPropagation();
        }}
        onMouseDown={(event: React.MouseEvent<HTMLButtonElement, MouseEvent>) => {
          event.preventDefault();
        }}
      >
        <Switch
          id="code-editor-darkmode-switch"
          aria-label="darkmode-switch"
          label="Switch to Light Theme"
          labelOff="Switch to Dark Theme"
          isChecked={isEditorDarkMode}
          onChange={(_event: React.FormEvent<HTMLInputElement>, checked: boolean) => {
            setIsEditorDarkMode(checked);
          }}
        />
      </Button>
    </div>
  );

  const defaultCodeTemplate0 = (
    <CodeEditorControl
      icon={<CodeIcon/>}
      aria-label="Execute code"
      tooltipProps={{content: 'Sample Code #1'}}
      onClick={() => {
        setCode(
          `a = 1
print("a = %d" % a)`,
        );
      }}
    />
  );

  const defaultCodeTemplate1 = (
    <CodeEditorControl
      icon={<CodeIcon/>}
      aria-label="Execute code"
      tooltipProps={{content: 'Sample Code #1'}}
      onClick={() => {
        setCode(
          `a = a + 1
print("a = %d" % a)`,
        );
      }}
    />
  );

  const defaultCodeTemplate2 = (
    <CodeEditorControl
      icon={<CodeIcon/>}
      aria-label="Execute code"
      tooltipProps={{content: 'Sample Code #2'}}
      onClick={() => {
        setCode(
          `b = a * 2
print("a = %d, b = %d" % (a, b))`,
        );
      }}
    />
  );

  const defaultCodeTemplate3 = (
    <CodeEditorControl
      icon={<CodeIcon/>}
      aria-label="Execute code"
      tooltipProps={{content: 'Sample Code #3'}}
      onClick={() => {
        setCode(`c = (b + 15) * ((a - 2) * a)
print("a = %d, b = %d, c = %d" % (a, b, c))`);
      }}
    />
  );

  const defaultCodeTemplate4 = (
    <CodeEditorControl
      icon={<CodeIcon/>}
      aria-label="Execute code"
      tooltipProps={{content: 'Sample Code #4'}}
      onClick={() => {
        setCode(`a = 1
b = a + 2
c = (b * 3) - a
d = (2 * a) - (4 * b) + (3 + c)
e = (d ^ 2) + c + b + a
f = e + d + c + b + a
g = f + e + d + c + b + a
h = g + f + e + d + c + b + a
i = h + g + f + e + d + c + b + a
j = i + h + g + f + e + d + c + b + a
print(f"i = {i}")
`);
      }}
    />
  );

  const getCustomControls = () => {
    if (props.showCodeTemplates) {
      return [defaultCodeTemplate0, defaultCodeTemplate1, defaultCodeTemplate2, defaultCodeTemplate3, defaultCodeTemplate4, fileNameField, darkLightThemeSwitch];
    } else {
      return [fileNameField, darkLightThemeSwitch];
    }
  }

  const getDownloadFilename = () => {
    if (!filename || filename.length == 0) {
      return defaultFilename;
    }

    const fileExtension: string = CodeEditor.getExtensionFromLanguage(props.language);
    if (filename.endsWith(`.${fileExtension}`)) {
      const filenameLength: number = filename.length;
      const extensionLength: number = fileExtension.length + 1; // +1 for the period.
      return filename.substring(0, filenameLength - extensionLength);
    }

    return filename;
  }

  return (
    <CodeEditor
      isDarkTheme={isEditorDarkMode}
      shortcutsPopoverProps={shortcutsPopoverProps}
      customControls={getCustomControls()}
      isLanguageLabelVisible
      isUploadEnabled
      downloadFileName={getDownloadFilename()}
      isDownloadEnabled
      isCopyEnabled
      code={code}
      /* eslint-disable-next-line @typescript-eslint/no-unused-vars */
      onChange={(value: string, _: editor.IModelContentChangedEvent) => {
        setCode(value);
      }}
      language={props.language}
      onEditorDidMount={onEditorDidMount}
      height={`${props.height}px`}
    />
  );
};
