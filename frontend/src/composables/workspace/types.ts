export type User = { id: number; username: string; email: string };

export type PicbedConfig = {
  id: number;
  picbed_type: string;
  config_name: string;
  is_default: boolean;
  created_at: string;
  updated_at: string;
  config?: Record<string, string | boolean>;
};

export type ConversionRecord = {
  id: number;
  original_filename: string;
  source_picbed: string;
  target_picbed: string;
  status: string;
  error_message?: string;
  image_count: number;
  created_at: string;
};

export type MarkdownImage = { raw: string; url: string; alt: string; picbed: string };

export type ConfigField = {
  key: string;
  label: string;
  placeholder: string;
  required: boolean;
  secret: boolean;
};

export type PicbedTypeDef = { value: string; label: string; description: string; fields: ConfigField[] };

export type BatchFile = {
  id: string;
  filename: string;
  content: string;
  images: MarkdownImage[];
  convertedContent: string;
  changed: number;
  status: 'ready' | 'analyzed' | 'success' | 'failed';
  error: string;
};

export type LocalDocument = {
  id: string;
  filename: string;
  content: string;
  references: MarkdownImage[];
  matched: number;
  missing: string[];
  convertedContent: string;
  changed: number;
  status: 'ready' | 'analyzed' | 'success' | 'failed';
  error: string;
};

export type LocalImageFile = {
  key: string;
  name: string;
  path: string;
  file: File;
};

export type RequestError = Error & { status?: number };

export type WorkspaceTab = 'convert' | 'localUpload' | 'configs' | 'records';
