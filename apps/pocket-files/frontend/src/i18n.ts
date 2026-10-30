// Centralized internationalization for Pocket Files.
// Every user-visible string lives here. Components must not hardcode text.
// A key-parity assertion runs at module load and throws if any language
// drifts from the canonical `en` key set.

export interface Strings {
  // Branding / common
  appName: string;
  appTagline: string;
  commonBack: string;
  commonCancel: string;
  commonConfirm: string;
  commonClose: string;
  commonLoading: string;
  commonError: string;
  commonUnavailable: string;
  commonNotAvailable: string;
  commonRetry: string;
  commonEmpty: string;
  commonEmptyFolder: string;
  commonYes: string;
  commonNo: string;
  commonSearch: string;
  commonMock: string;
  commonMockStorage: string;
  commonMockData: string;
  commonItems: string;
  commonSelected: string;
  commonResults: string;

  // Sidebar / navigation
  navStorage: string;
  navLibraries: string;
  navOverview: string;
  navFiles: string;
  navTransfers: string;
  navFavorites: string;
  navRecents: string;
  navSettings: string;
  accountName: string;
  accountProfile: string;
  sidebarCollapse: string;
  sidebarExpand: string;

  // Overview
  overviewEyebrow: string;
  overviewTitle: string;
  overviewSubtitle: string;
  overviewBrowse: string;
  overviewQuickAccess: string;
  overviewQuickAccessSub: string;
  overviewEditLocations: string;
  overviewNoticeTitle: string;
  overviewNoticeBody: string;
  overviewOpen: string;

  // Device cards
  deviceSystem: string;
  deviceRemovableSd: string;
  deviceRemovableUsb: string;
  deviceOther: string;
  deviceReady: string;
  deviceUsed: string;
  deviceTotal: string;
  deviceFree: string;
  deviceOpen: string;
  deviceReadOnly: string;

  // Files
  filesTitle: string;
  filesBack: string;
  filesNewFolder: string;
  filesName: string;
  filesSize: string;
  filesModified: string;
  filesType: string;
  filesSort: string;
  filesSortName: string;
  filesSortSize: string;
  filesSortModified: string;
  filesSortType: string;
  filesAsc: string;
  filesDesc: string;
  filesFilterType: string;
  filesFilterAll: string;
  filesFilterFolders: string;
  filesFilterFiles: string;
  filesPage: string;
  filesPrev: string;
  filesNext: string;
  filesSelect: string;
  filesDone: string;
  filesBulkCopy: string;
  filesBulkMove: string;
  filesBulkDelete: string;
  filesDestination: string;
  filesConfirmDeleteTitle: string;
  filesConfirmDeleteBody: string;
  filesConfirmDelete: string;
  filesOperationDone: string;
  filesOperationFailed: string;
  filesUnavailable: string;

  // Storage detail
  storageTitle: string;
  storageUsage: string;
  storageUsed: string;
  storageTotal: string;
  storageFree: string;
  storageBrowse: string;
  storageUnavailable: string;
  storageKindInternal: string;
  storageKindTf: string;
  storageKindUsb: string;
  storageKindOther: string;
  storageReadOnly: string;
  storageMockNote: string;

  // Transfers
  transfersTitle: string;
  transfersEmpty: string;
  transfersNone: string;
  transfersPause: string;
  transfersResume: string;
  transfersCancel: string;
  transfersRetry: string;
  transfersStatusQueued: string;
  transfersStatusActive: string;
  transfersStatusPaused: string;
  transfersStatusCompleted: string;
  transfersStatusFailed: string;
  transfersStatusCanceled: string;
  transfersUnavailable: string;
  transfersProgress: string;

  // Search
  searchTitle: string;
  searchPlaceholder: string;
  searchEmpty: string;
  searchSubmit: string;
  searchType: string;
  searchMinSize: string;
  searchMaxSize: string;
  searchUnavailable: string;

  // Favorites
  favoritesTitle: string;
  favoritesEmpty: string;
  favoritesNone: string;
  favoritesRemove: string;
  favoritesAdd: string;
  favoritesUnavailable: string;

  // Recents
  recentsTitle: string;
  recentsEmpty: string;
  recentsNone: string;
  recentsKind: string;
  recentsKindAll: string;
  recentsKindFile: string;
  recentsKindFolder: string;
  recentsUnavailable: string;

  // Libraries
  librariesTitle: string;
  librariesBooks: string;
  librariesMusic: string;
  librariesVideos: string;
  librariesGames: string;
  librariesDownloads: string;
  librariesOpen: string;
  librariesUnavailable: string;
  librariesNotImplemented: string;

  // Analytics
  analyticsTitle: string;
  analyticsUnavailable: string;
  analyticsSummary: string;
  analyticsByKind: string;
  analyticsByExtension: string;
  analyticsFiles: string;
  analyticsFolders: string;
  analyticsTotalSize: string;
  analyticsLargest: string;

  // Settings
  settingsTitle: string;
  settingsSubtitle: string;
  settingsPreferences: string;
  settingsAppearance: string;
  settingsTheme: string;
  settingsLanguage: string;
  settingsProfile: string;
  settingsProfileAuto: string;
  settingsThemeSystem: string;
  settingsThemeLight: string;
  settingsThemeDark: string;
  settingsThemeOled: string;
  settingsLangEn: string;
  settingsLangZh: string;
  settingsLangJa: string;
  settingsProfileDesktop: string;
  settingsProfileHandheld: string;
  settingsProfileReader: string;
  settingsProfileDap: string;
}

export const en: Strings = {
  appName: 'Pocket Files',
  appTagline: 'Storage Manager',
  commonBack: 'Back',
  commonCancel: 'Cancel',
  commonConfirm: 'Confirm',
  commonClose: 'Close',
  commonLoading: 'Loading…',
  commonError: 'Something went wrong',
  commonUnavailable: 'Unavailable',
  commonNotAvailable: 'Not available',
  commonRetry: 'Retry',
  commonEmpty: 'Nothing here yet',
  commonEmptyFolder: 'This folder is empty',
  commonYes: 'Yes',
  commonNo: 'No',
  commonSearch: 'Search files and folders',
  commonMock: 'MOCK',
  commonMockStorage: 'MOCK STORAGE',
  commonMockData: 'MOCK DATA',
  commonItems: '{n} items',
  commonSelected: '{n} selected',
  commonResults: '{n} results',

  navStorage: 'STORAGE',
  navLibraries: 'LIBRARIES',
  navOverview: 'Overview',
  navFiles: 'Files',
  navTransfers: 'Transfers',
  navFavorites: 'Favorites',
  navRecents: 'Recents',
  navSettings: 'Settings',
  accountName: 'Pocket User',
  accountProfile: 'Local profile',
  sidebarCollapse: 'Collapse sidebar',
  sidebarExpand: 'Expand sidebar',

  overviewEyebrow: 'YOUR DEVICES',
  overviewTitle: 'Storage Overview',
  overviewSubtitle: 'Manage files across all your connected storage.',
  overviewBrowse: 'Browse files',
  overviewQuickAccess: 'Quick access',
  overviewQuickAccessSub: 'Your frequently used locations',
  overviewEditLocations: 'Edit locations',
  overviewNoticeTitle: 'Preview mode',
  overviewNoticeBody: 'These are temporary mock storage devices. Connect real storage to browse your files.',
  overviewOpen: 'Open',

  deviceSystem: 'System storage',
  deviceRemovableSd: 'Removable · microSD',
  deviceRemovableUsb: 'Removable · USB',
  deviceOther: 'Storage device',
  deviceReady: 'Ready',
  deviceUsed: 'used',
  deviceTotal: 'Total',
  deviceFree: 'Free',
  deviceOpen: 'Open',
  deviceReadOnly: 'Read-only',

  filesTitle: 'Files',
  filesBack: 'Back',
  filesNewFolder: 'New folder',
  filesName: 'Name',
  filesSize: 'Size',
  filesModified: 'Modified',
  filesType: 'Type',
  filesSort: 'Sort',
  filesSortName: 'Name',
  filesSortSize: 'Size',
  filesSortModified: 'Modified',
  filesSortType: 'Type',
  filesAsc: 'Ascending',
  filesDesc: 'Descending',
  filesFilterType: 'Filter type',
  filesFilterAll: 'All',
  filesFilterFolders: 'Folders',
  filesFilterFiles: 'Files',
  filesPage: 'Page',
  filesPrev: 'Previous',
  filesNext: 'Next',
  filesSelect: 'Select',
  filesDone: 'Done',
  filesBulkCopy: 'Copy',
  filesBulkMove: 'Move',
  filesBulkDelete: 'Delete',
  filesDestination: 'Destination folder',
  filesConfirmDeleteTitle: 'Delete items?',
  filesConfirmDeleteBody: 'This will permanently delete {n} selected item(s). This cannot be undone.',
  filesConfirmDelete: 'Delete',
  filesOperationDone: 'Operation completed',
  filesOperationFailed: 'Operation failed',
  filesUnavailable: 'Files unavailable',

  storageTitle: 'Storage',
  storageUsage: 'Storage Usage',
  storageUsed: 'Used',
  storageTotal: 'Total',
  storageFree: 'Free',
  storageBrowse: 'Browse files',
  storageUnavailable: 'Storage usage unavailable',
  storageKindInternal: 'System storage',
  storageKindTf: 'Removable · microSD',
  storageKindUsb: 'Removable · USB',
  storageKindOther: 'Storage device',
  storageReadOnly: 'Read-only',
  storageMockNote: 'Mock storage device',

  transfersTitle: 'Transfers',
  transfersEmpty: 'No transfers in progress',
  transfersNone: 'Nothing here yet',
  transfersPause: 'Pause',
  transfersResume: 'Resume',
  transfersCancel: 'Cancel',
  transfersRetry: 'Retry',
  transfersStatusQueued: 'Queued',
  transfersStatusActive: 'Active',
  transfersStatusPaused: 'Paused',
  transfersStatusCompleted: 'Completed',
  transfersStatusFailed: 'Failed',
  transfersStatusCanceled: 'Canceled',
  transfersUnavailable: 'Transfer list unavailable',
  transfersProgress: '{n}%',

  searchTitle: 'Search',
  searchPlaceholder: 'Search files and folders',
  searchEmpty: 'No results found',
  searchSubmit: 'Search',
  searchType: 'Type',
  searchMinSize: 'Min size',
  searchMaxSize: 'Max size',
  searchUnavailable: 'Search unavailable',

  favoritesTitle: 'Favorites',
  favoritesEmpty: 'Your pinned files and folders',
  favoritesNone: 'Nothing here yet',
  favoritesRemove: 'Remove',
  favoritesAdd: 'Add to favorites',
  favoritesUnavailable: 'Favorites unavailable',

  recentsTitle: 'Recents',
  recentsEmpty: 'Files you opened recently',
  recentsNone: 'Nothing here yet',
  recentsKind: 'Kind',
  recentsKindAll: 'All',
  recentsKindFile: 'Files',
  recentsKindFolder: 'Folders',
  recentsUnavailable: 'Recents unavailable',

  librariesTitle: 'Libraries',
  librariesBooks: 'Books',
  librariesMusic: 'Music',
  librariesVideos: 'Videos',
  librariesGames: 'Games',
  librariesDownloads: 'Downloads',
  librariesOpen: 'Open',
  librariesUnavailable: 'Virtual libraries unavailable',
  librariesNotImplemented: 'This library view is not available yet',

  analyticsTitle: 'Storage Analytics',
  analyticsUnavailable: 'Analytics unavailable',
  analyticsSummary: 'Summary',
  analyticsByKind: 'By kind',
  analyticsByExtension: 'By extension',
  analyticsFiles: 'Files',
  analyticsFolders: 'Folders',
  analyticsTotalSize: 'Total size',
  analyticsLargest: 'Largest files',

  settingsTitle: 'Settings',
  settingsSubtitle: 'Customize Pocket Files for your device.',
  settingsPreferences: 'PREFERENCES',
  settingsAppearance: 'Appearance',
  settingsTheme: 'Color theme',
  settingsLanguage: 'Language',
  settingsProfile: 'Device profile',
  settingsProfileAuto: 'Auto',
  settingsThemeSystem: 'System',
  settingsThemeLight: 'Light',
  settingsThemeDark: 'Dark',
  settingsThemeOled: 'OLED',
  settingsLangEn: 'English',
  settingsLangZh: '简体中文',
  settingsLangJa: '日本語',
  settingsProfileDesktop: 'Desktop',
  settingsProfileHandheld: 'Handheld',
  settingsProfileReader: 'Reader',
  settingsProfileDap: 'DAP',
};

export const zh: Strings = {
  appName: 'Pocket Files',
  appTagline: '存储管理器',
  commonBack: '返回',
  commonCancel: '取消',
  commonConfirm: '确认',
  commonClose: '关闭',
  commonLoading: '加载中…',
  commonError: '出错了',
  commonUnavailable: '不可用',
  commonNotAvailable: '暂不可用',
  commonRetry: '重试',
  commonEmpty: '这里还什么都没有',
  commonEmptyFolder: '此文件夹为空',
  commonYes: '是',
  commonNo: '否',
  commonSearch: '搜索文件和文件夹',
  commonMock: '模拟',
  commonMockStorage: '模拟存储',
  commonMockData: '模拟数据',
  commonItems: '{n} 个项目',
  commonSelected: '已选择 {n} 项',
  commonResults: '{n} 条结果',

  navStorage: '存储设备',
  navLibraries: '资料库',
  navOverview: '概览',
  navFiles: '文件',
  navTransfers: '传输',
  navFavorites: '收藏',
  navRecents: '最近',
  navSettings: '设置',
  accountName: 'Pocket 用户',
  accountProfile: '本地账户',
  sidebarCollapse: '收起侧边栏',
  sidebarExpand: '展开侧边栏',

  overviewEyebrow: '你的设备',
  overviewTitle: '存储概览',
  overviewSubtitle: '管理你所有已连接存储中的文件。',
  overviewBrowse: '浏览文件',
  overviewQuickAccess: '快速访问',
  overviewQuickAccessSub: '你常用的位置',
  overviewEditLocations: '编辑位置',
  overviewNoticeTitle: '预览模式',
  overviewNoticeBody: '这些是临时的模拟存储设备。连接真实存储后即可浏览你的文件。',
  overviewOpen: '打开',

  deviceSystem: '系统存储',
  deviceRemovableSd: '可移动 · microSD',
  deviceRemovableUsb: '可移动 · USB',
  deviceOther: '存储设备',
  deviceReady: '就绪',
  deviceUsed: '已用',
  deviceTotal: '总计',
  deviceFree: '可用',
  deviceOpen: '打开',
  deviceReadOnly: '只读',

  filesTitle: '文件',
  filesBack: '返回',
  filesNewFolder: '新建文件夹',
  filesName: '名称',
  filesSize: '大小',
  filesModified: '修改时间',
  filesType: '类型',
  filesSort: '排序',
  filesSortName: '名称',
  filesSortSize: '大小',
  filesSortModified: '修改时间',
  filesSortType: '类型',
  filesAsc: '升序',
  filesDesc: '降序',
  filesFilterType: '筛选类型',
  filesFilterAll: '全部',
  filesFilterFolders: '文件夹',
  filesFilterFiles: '文件',
  filesPage: '页',
  filesPrev: '上一页',
  filesNext: '下一页',
  filesSelect: '选择',
  filesDone: '完成',
  filesBulkCopy: '复制',
  filesBulkMove: '移动',
  filesBulkDelete: '删除',
  filesDestination: '目标文件夹',
  filesConfirmDeleteTitle: '删除项目？',
  filesConfirmDeleteBody: '这将永久删除选中的 {n} 个项目，且无法撤销。',
  filesConfirmDelete: '删除',
  filesOperationDone: '操作已完成',
  filesOperationFailed: '操作失败',
  filesUnavailable: '文件不可用',

  storageTitle: '存储',
  storageUsage: '存储用量',
  storageUsed: '已用',
  storageTotal: '总计',
  storageFree: '可用',
  storageBrowse: '浏览文件',
  storageUnavailable: '存储用量不可用',
  storageKindInternal: '系统存储',
  storageKindTf: '可移动 · microSD',
  storageKindUsb: '可移动 · USB',
  storageKindOther: '存储设备',
  storageReadOnly: '只读',
  storageMockNote: '模拟存储设备',

  transfersTitle: '传输',
  transfersEmpty: '当前没有进行中的传输',
  transfersNone: '这里还什么都没有',
  transfersPause: '暂停',
  transfersResume: '继续',
  transfersCancel: '取消',
  transfersRetry: '重试',
  transfersStatusQueued: '排队中',
  transfersStatusActive: '进行中',
  transfersStatusPaused: '已暂停',
  transfersStatusCompleted: '已完成',
  transfersStatusFailed: '失败',
  transfersStatusCanceled: '已取消',
  transfersUnavailable: '传输列表不可用',
  transfersProgress: '{n}%',

  searchTitle: '搜索',
  searchPlaceholder: '搜索文件和文件夹',
  searchEmpty: '没有找到结果',
  searchSubmit: '搜索',
  searchType: '类型',
  searchMinSize: '最小大小',
  searchMaxSize: '最大大小',
  searchUnavailable: '搜索不可用',

  favoritesTitle: '收藏',
  favoritesEmpty: '你固定的文件和文件夹',
  favoritesNone: '这里还什么都没有',
  favoritesRemove: '移除',
  favoritesAdd: '添加到收藏',
  favoritesUnavailable: '收藏不可用',

  recentsTitle: '最近',
  recentsEmpty: '你最近打开过的文件',
  recentsNone: '这里还什么都没有',
  recentsKind: '类型',
  recentsKindAll: '全部',
  recentsKindFile: '文件',
  recentsKindFolder: '文件夹',
  recentsUnavailable: '最近不可用',

  librariesTitle: '资料库',
  librariesBooks: '图书',
  librariesMusic: '音乐',
  librariesVideos: '视频',
  librariesGames: '游戏',
  librariesDownloads: '下载',
  librariesOpen: '打开',
  librariesUnavailable: '虚拟资料库不可用',
  librariesNotImplemented: '该资料库视图暂不可用',

  analyticsTitle: '存储分析',
  analyticsUnavailable: '分析不可用',
  analyticsSummary: '概览',
  analyticsByKind: '按类型',
  analyticsByExtension: '按扩展名',
  analyticsFiles: '文件',
  analyticsFolders: '文件夹',
  analyticsTotalSize: '总大小',
  analyticsLargest: '最大文件',

  settingsTitle: '设置',
  settingsSubtitle: '为你的设备定制 Pocket Files。',
  settingsPreferences: '偏好设置',
  settingsAppearance: '外观',
  settingsTheme: '配色主题',
  settingsLanguage: '语言',
  settingsProfile: '设备类型',
  settingsProfileAuto: '自动',
  settingsThemeSystem: '跟随系统',
  settingsThemeLight: '浅色',
  settingsThemeDark: '深色',
  settingsThemeOled: 'OLED',
  settingsLangEn: 'English',
  settingsLangZh: '简体中文',
  settingsLangJa: '日本語',
  settingsProfileDesktop: '桌面',
  settingsProfileHandheld: '手持',
  settingsProfileReader: '阅读器',
  settingsProfileDap: '播放器',
};

export const ja: Strings = {
  appName: 'Pocket Files',
  appTagline: 'ストレージマネージャー',
  commonBack: '戻る',
  commonCancel: 'キャンセル',
  commonConfirm: '確認',
  commonClose: '閉じる',
  commonLoading: '読み込み中…',
  commonError: '問題が発生しました',
  commonUnavailable: '利用不可',
  commonNotAvailable: '現在利用できません',
  commonRetry: '再試行',
  commonEmpty: 'まだ何もありません',
  commonEmptyFolder: 'このフォルダは空です',
  commonYes: 'はい',
  commonNo: 'いいえ',
  commonSearch: 'ファイルとフォルダを検索',
  commonMock: 'モック',
  commonMockStorage: 'モックストレージ',
  commonMockData: 'モックデータ',
  commonItems: '項目 {n} 件',
  commonSelected: '{n} 件選択中',
  commonResults: '結果 {n} 件',

  navStorage: 'ストレージ',
  navLibraries: 'ライブラリ',
  navOverview: '概要',
  navFiles: 'ファイル',
  navTransfers: '転送',
  navFavorites: 'お気に入り',
  navRecents: '最近',
  navSettings: '設定',
  accountName: 'Pocket ユーザー',
  accountProfile: 'ローカルプロフィール',
  sidebarCollapse: 'サイドバーを折りたたむ',
  sidebarExpand: 'サイドバーを展開',

  overviewEyebrow: 'あなたのデバイス',
  overviewTitle: 'ストレージ概要',
  overviewSubtitle: '接続されたすべてのストレージのファイルを管理します。',
  overviewBrowse: 'ファイルを表示',
  overviewQuickAccess: 'クイックアクセス',
  overviewQuickAccessSub: 'よく使う場所',
  overviewEditLocations: '場所を編集',
  overviewNoticeTitle: 'プレビューモード',
  overviewNoticeBody: 'これらは一時的なモックストレージデバイスです。実際のストレージを接続するとファイルを閲覧できます。',
  overviewOpen: '開く',

  deviceSystem: 'システムストレージ',
  deviceRemovableSd: 'リムーバブル · microSD',
  deviceRemovableUsb: 'リムーバブル · USB',
  deviceOther: 'ストレージデバイス',
  deviceReady: '準備完了',
  deviceUsed: '使用済み',
  deviceTotal: '合計',
  deviceFree: '空き',
  deviceOpen: '開く',
  deviceReadOnly: '読み取り専用',

  filesTitle: 'ファイル',
  filesBack: '戻る',
  filesNewFolder: '新規フォルダ',
  filesName: '名前',
  filesSize: 'サイズ',
  filesModified: '更新日時',
  filesType: '種類',
  filesSort: '並べ替え',
  filesSortName: '名前',
  filesSortSize: 'サイズ',
  filesSortModified: '更新日時',
  filesSortType: '種類',
  filesAsc: '昇順',
  filesDesc: '降順',
  filesFilterType: '種類で絞り込み',
  filesFilterAll: 'すべて',
  filesFilterFolders: 'フォルダ',
  filesFilterFiles: 'ファイル',
  filesPage: 'ページ',
  filesPrev: '前へ',
  filesNext: '次へ',
  filesSelect: '選択',
  filesDone: '完了',
  filesBulkCopy: 'コピー',
  filesBulkMove: '移動',
  filesBulkDelete: '削除',
  filesDestination: '移動先フォルダ',
  filesConfirmDeleteTitle: '項目を削除しますか？',
  filesConfirmDeleteBody: '選択した {n} 個の項目を完全に削除します。元に戻せません。',
  filesConfirmDelete: '削除',
  filesOperationDone: '操作が完了しました',
  filesOperationFailed: '操作に失敗しました',
  filesUnavailable: 'ファイルを利用できません',

  storageTitle: 'ストレージ',
  storageUsage: 'ストレージ使用量',
  storageUsed: '使用済み',
  storageTotal: '合計',
  storageFree: '空き',
  storageBrowse: 'ファイルを表示',
  storageUnavailable: 'ストレージ使用量を利用できません',
  storageKindInternal: 'システムストレージ',
  storageKindTf: 'リムーバブル · microSD',
  storageKindUsb: 'リムーバブル · USB',
  storageKindOther: 'ストレージデバイス',
  storageReadOnly: '読み取り専用',
  storageMockNote: 'モックストレージデバイス',

  transfersTitle: '転送',
  transfersEmpty: '進行中の転送はありません',
  transfersNone: 'まだ何もありません',
  transfersPause: '一時停止',
  transfersResume: '再開',
  transfersCancel: 'キャンセル',
  transfersRetry: '再試行',
  transfersStatusQueued: '待機中',
  transfersStatusActive: '進行中',
  transfersStatusPaused: '一時停止中',
  transfersStatusCompleted: '完了',
  transfersStatusFailed: '失敗',
  transfersStatusCanceled: 'キャンセル済み',
  transfersUnavailable: '転送リストを利用できません',
  transfersProgress: '{n}%',

  searchTitle: '検索',
  searchPlaceholder: 'ファイルとフォルダを検索',
  searchEmpty: '結果が見つかりませんでした',
  searchSubmit: '検索',
  searchType: '種類',
  searchMinSize: '最小サイズ',
  searchMaxSize: '最大サイズ',
  searchUnavailable: '検索を利用できません',

  favoritesTitle: 'お気に入り',
  favoritesEmpty: 'ピン留めしたファイルとフォルダ',
  favoritesNone: 'まだ何もありません',
  favoritesRemove: '削除',
  favoritesAdd: 'お気に入りに追加',
  favoritesUnavailable: 'お気に入りを利用できません',

  recentsTitle: '最近',
  recentsEmpty: '最近開いたファイル',
  recentsNone: 'まだ何もありません',
  recentsKind: '種類',
  recentsKindAll: 'すべて',
  recentsKindFile: 'ファイル',
  recentsKindFolder: 'フォルダ',
  recentsUnavailable: '最近を利用できません',

  librariesTitle: 'ライブラリ',
  librariesBooks: '本',
  librariesMusic: '音楽',
  librariesVideos: 'ビデオ',
  librariesGames: 'ゲーム',
  librariesDownloads: 'ダウンロード',
  librariesOpen: '開く',
  librariesUnavailable: '仮想ライブラリを利用できません',
  librariesNotImplemented: 'このライブラリビューはまだ利用できません',

  analyticsTitle: 'ストレージ分析',
  analyticsUnavailable: '分析を利用できません',
  analyticsSummary: '概要',
  analyticsByKind: '種類別',
  analyticsByExtension: '拡張子別',
  analyticsFiles: 'ファイル',
  analyticsFolders: 'フォルダ',
  analyticsTotalSize: '合計サイズ',
  analyticsLargest: '大きいファイル',

  settingsTitle: '設定',
  settingsSubtitle: 'デバイスに合わせて Pocket Files をカスタマイズします。',
  settingsPreferences: '環境設定',
  settingsAppearance: '外観',
  settingsTheme: 'カラーテーマ',
  settingsLanguage: '言語',
  settingsProfile: 'デバイスタイプ',
  settingsProfileAuto: '自動',
  settingsThemeSystem: 'システム',
  settingsThemeLight: 'ライト',
  settingsThemeDark: 'ダーク',
  settingsThemeOled: 'OLED',
  settingsLangEn: 'English',
  settingsLangZh: '简体中文',
  settingsLangJa: '日本語',
  settingsProfileDesktop: 'デスクトップ',
  settingsProfileHandheld: 'ハンドヘルド',
  settingsProfileReader: 'リーダー',
  settingsProfileDap: 'DAP',
};

export const dictionaries = { en, zh, ja } as const;

export type Lang = keyof typeof dictionaries;

export const langNames: Record<Lang, string> = {
  en: 'English',
  zh: '简体中文',
  ja: '日本語',
};

// Build-time / load-time key-parity assertion. Throws immediately if any
// language is missing keys present in `en`, or defines keys `en` lacks.
(() => {
  const base = Object.keys(en).sort();
  const baseSet = new Set(base);
  for (const code of Object.keys(dictionaries) as Lang[]) {
    const keys = Object.keys(dictionaries[code]).sort();
    const missing = base.filter((k) => !keys.includes(k));
    const extra = keys.filter((k) => !baseSet.has(k));
    if (missing.length > 0 || extra.length > 0) {
      throw new Error(
        `i18n key parity failure for "${code}": missing=[${missing.join(', ')}] extra=[${extra.join(', ')}]`,
      );
    }
  }
})();

export type TranslateParams = Record<string, string | number>;

export function translate(
  lang: Lang,
  key: keyof Strings,
  params?: TranslateParams,
): string {
  const dict = dictionaries[lang] ?? en;
  let value = dict[key] ?? en[key] ?? (key as string);
  if (params) {
    for (const [name, val] of Object.entries(params)) {
      value = value.replace(new RegExp(`\\{${name}\\}`, 'g'), String(val));
    }
  }
  return value;
}
