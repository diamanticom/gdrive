package main

import (
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"path/filepath"
	"time"

	"github.com/gdrive-org/gdrive/reconciler"

	"github.com/gdrive-org/gdrive/utils"

	"github.com/gdrive-org/gdrive/cli"
	"github.com/gdrive-org/gdrive/drive"
)

func listHandler(ctx cli.Context) {
	args := ctx.Args()
	owner, ok := os.LookupEnv(utils.AssetOwnerKey)
	var query string
	if !ok {
		query = args.String("query")
	} else {
		query = fmt.Sprintf("trashed = false and '%s' in owners", owner)
	}

	err := utils.NewDrive(args).List(drive.ListFilesArgs{
		Out:         os.Stdout,
		MaxFiles:    args.Int64("maxFiles"),
		NameWidth:   args.Int64("nameWidth"),
		Query:       query,
		SortOrder:   args.String("sortOrder"),
		SkipHeader:  args.Bool("skipHeader"),
		SizeInBytes: args.Bool("sizeInBytes"),
		AbsPath:     args.Bool("absPath"),
		JsonOut:     args.Bool("jsonOut"),
	})
	utils.CheckErr(err)
}

func listChangesHandler(ctx cli.Context) {
	args := ctx.Args()
	err := utils.NewDrive(args).ListChanges(drive.ListChangesArgs{
		Out:        os.Stdout,
		PageToken:  args.String("pageToken"),
		MaxChanges: args.Int64("maxChanges"),
		Now:        args.Bool("now"),
		NameWidth:  args.Int64("nameWidth"),
		SkipHeader: args.Bool("skipHeader"),
		JsonOut:    args.Bool("jsonOut"),
	})
	utils.CheckErr(err)
}

func downloadHandler(ctx cli.Context) {
	args := ctx.Args()
	checkDownloadArgs(args)
	err := utils.NewDrive(args).Download(drive.DownloadArgs{
		Out:       os.Stdout,
		Id:        args.String("fileId"),
		Force:     args.Bool("force"),
		Skip:      args.Bool("skip"),
		Path:      args.String("path"),
		Delete:    args.Bool("delete"),
		Recursive: args.Bool("recursive"),
		Stdout:    args.Bool("stdout"),
		Progress:  progressWriter(args.Bool("noProgress")),
		Timeout:   durationInSeconds(args.Int64("timeout")),
		JsonOut:   args.Bool("jsonOut"),
	})
	utils.CheckErr(err)
}

func downloadQueryHandler(ctx cli.Context) {
	args := ctx.Args()
	err := utils.NewDrive(args).DownloadQuery(drive.DownloadQueryArgs{
		Out:       os.Stdout,
		Query:     args.String("query"),
		Force:     args.Bool("force"),
		Skip:      args.Bool("skip"),
		Recursive: args.Bool("recursive"),
		Path:      args.String("path"),
		Progress:  progressWriter(args.Bool("noProgress")),
		JsonOut:   args.Bool("jsonOut"),
	})
	utils.CheckErr(err)
}

func downloadSyncHandler(ctx cli.Context) {
	args := ctx.Args()
	cachePath := filepath.Join(args.String("configDir"), utils.DefaultCacheFileName)
	err := utils.NewDrive(args).DownloadSync(drive.DownloadSyncArgs{
		Out:              os.Stdout,
		Progress:         progressWriter(args.Bool("noProgress")),
		Path:             args.String("path"),
		RootId:           args.String("fileId"),
		DryRun:           args.Bool("dryRun"),
		DeleteExtraneous: args.Bool("deleteExtraneous"),
		Timeout:          durationInSeconds(args.Int64("timeout")),
		Resolution:       conflictResolution(args),
		Comparer:         NewCachedMd5Comparer(cachePath),
	})
	utils.CheckErr(err)
}

func downloadRevisionHandler(ctx cli.Context) {
	args := ctx.Args()
	err := utils.NewDrive(args).DownloadRevision(drive.DownloadRevisionArgs{
		Out:        os.Stdout,
		FileId:     args.String("fileId"),
		RevisionId: args.String("revId"),
		Force:      args.Bool("force"),
		Stdout:     args.Bool("stdout"),
		Path:       args.String("path"),
		Progress:   progressWriter(args.Bool("noProgress")),
		Timeout:    durationInSeconds(args.Int64("timeout")),
		JsonOut:    args.Bool("jsonOut"),
	})
	utils.CheckErr(err)
}

func specApplyHandler(ctx cli.Context) {
	args := ctx.Args()
	fileName := args.String("specFile")
	g := utils.NewDrive(args)

	r, err := reconciler.New(fileName, g)
	if err != nil {
		utils.CheckErr(err)
	}

	if err := r.Reconcile(); err != nil {
		utils.CheckErr(err)
	}
}

func specGenHandler(ctx cli.Context) {
	args := ctx.Args()
	g := utils.NewDrive(args)

	s := new(reconciler.Spec)
	s.SetDriver(g)

	if err := s.Generate(); err != nil {
		utils.CheckErr(err)
	}
}

func uploadHandler(ctx cli.Context) {
	args := ctx.Args()
	checkUploadArgs(args)
	err := utils.NewDrive(args).Upload(drive.UploadArgs{
		Out:         os.Stdout,
		Progress:    progressWriter(args.Bool("noProgress")),
		Path:        args.String("path"),
		Name:        args.String("name"),
		Description: args.String("description"),
		Parents:     args.StringSlice("parent"),
		Mime:        args.String("mime"),
		Recursive:   args.Bool("recursive"),
		Share:       args.Bool("share"),
		Delete:      args.Bool("delete"),
		ChunkSize:   args.Int64("chunksize"),
		Timeout:     durationInSeconds(args.Int64("timeout")),
		JsonOut:     args.Bool("jsonOut"),
	})
	utils.CheckErr(err)
}

func uploadStdinHandler(ctx cli.Context) {
	args := ctx.Args()
	err := utils.NewDrive(args).UploadStream(drive.UploadStreamArgs{
		Out:         os.Stdout,
		In:          os.Stdin,
		Name:        args.String("name"),
		Description: args.String("description"),
		Parents:     args.StringSlice("parent"),
		Mime:        args.String("mime"),
		Share:       args.Bool("share"),
		ChunkSize:   args.Int64("chunksize"),
		Timeout:     durationInSeconds(args.Int64("timeout")),
		Progress:    progressWriter(args.Bool("noProgress")),
		JsonOut:     args.Bool("jsonOut"),
	})
	utils.CheckErr(err)
}

func uploadSyncHandler(ctx cli.Context) {
	args := ctx.Args()
	cachePath := filepath.Join(args.String("configDir"), utils.DefaultCacheFileName)
	err := utils.NewDrive(args).UploadSync(drive.UploadSyncArgs{
		Out:              os.Stdout,
		Progress:         progressWriter(args.Bool("noProgress")),
		Path:             args.String("path"),
		RootId:           args.String("fileId"),
		DryRun:           args.Bool("dryRun"),
		DeleteExtraneous: args.Bool("deleteExtraneous"),
		ChunkSize:        args.Int64("chunksize"),
		Timeout:          durationInSeconds(args.Int64("timeout")),
		Resolution:       conflictResolution(args),
		Comparer:         NewCachedMd5Comparer(cachePath),
		JsonOut:          args.Bool("jsonOut"),
	})
	utils.CheckErr(err)
}

func updateHandler(ctx cli.Context) {
	args := ctx.Args()
	err := utils.NewDrive(args).Update(drive.UpdateArgs{
		Out:         os.Stdout,
		Id:          args.String("fileId"),
		Path:        args.String("path"),
		Name:        args.String("name"),
		Description: args.String("description"),
		Parents:     args.StringSlice("parent"),
		Mime:        args.String("mime"),
		Progress:    progressWriter(args.Bool("noProgress")),
		ChunkSize:   args.Int64("chunksize"),
		Timeout:     durationInSeconds(args.Int64("timeout")),
		JsonOut:     args.Bool("jsonOut"),
	})
	utils.CheckErr(err)
}

func infoHandler(ctx cli.Context) {
	args := ctx.Args()
	err := utils.NewDrive(args).Info(drive.FileInfoArgs{
		Out:         os.Stdout,
		Id:          args.String("fileId"),
		SizeInBytes: args.Bool("sizeInBytes"),
		JsonOut:     args.Bool("jsonOut"),
	})
	utils.CheckErr(err)
}

func importHandler(ctx cli.Context) {
	args := ctx.Args()
	err := utils.NewDrive(args).Import(drive.ImportArgs{
		Mime:     args.String("mime"),
		Out:      os.Stdout,
		Path:     args.String("path"),
		Parents:  args.StringSlice("parent"),
		Progress: progressWriter(args.Bool("noProgress")),
		JsonOut:  args.Bool("jsonOut"),
	})
	utils.CheckErr(err)
}

func exportHandler(ctx cli.Context) {
	args := ctx.Args()
	err := utils.NewDrive(args).Export(drive.ExportArgs{
		Out:        os.Stdout,
		Id:         args.String("fileId"),
		Mime:       args.String("mime"),
		PrintMimes: args.Bool("printMimes"),
		Force:      args.Bool("force"),
		JsonOut:    args.Bool("jsonOut"),
	})
	utils.CheckErr(err)
}

func listRevisionsHandler(ctx cli.Context) {
	args := ctx.Args()
	err := utils.NewDrive(args).ListRevisions(drive.ListRevisionsArgs{
		Out:         os.Stdout,
		Id:          args.String("fileId"),
		NameWidth:   args.Int64("nameWidth"),
		SizeInBytes: args.Bool("sizeInBytes"),
		SkipHeader:  args.Bool("skipHeader"),
		JsonOut:     args.Bool("jsonOut"),
	})
	utils.CheckErr(err)
}

func mkdirHandler(ctx cli.Context) {
	args := ctx.Args()
	err := utils.NewDrive(args).Mkdir(drive.MkdirArgs{
		Out:         os.Stdout,
		Name:        args.String("name"),
		Description: args.String("description"),
		Parents:     args.StringSlice("parent"),
		JsonOut:     args.Bool("jsonOut"),
	})
	utils.CheckErr(err)
}

func mkdirpHandler(ctx cli.Context) {
	args := ctx.Args()
	err := utils.NewDrive(args).Mkdirp(drive.MkdirArgs{
		Out:         os.Stdout,
		Name:        args.String("name"),
		Description: args.String("description"),
		Parents:     args.StringSlice("parent"),
		JsonOut:     args.Bool("jsonOut"),
	})
	utils.CheckErr(err)
}

func shareHandler(ctx cli.Context) {
	args := ctx.Args()
	err := utils.NewDrive(args).Share(drive.ShareArgs{
		Out:          os.Stdout,
		FileId:       args.String("fileId"),
		Role:         args.String("role"),
		Type:         args.String("type"),
		Email:        args.String("email"),
		Domain:       args.String("domain"),
		Discoverable: args.Bool("discoverable"),
		JsonOut:      args.Bool("jsonOut"),
	})
	utils.CheckErr(err)
}

func shareListHandler(ctx cli.Context) {
	args := ctx.Args()
	err := utils.NewDrive(args).ListPermissions(drive.ListPermissionsArgs{
		Out:     os.Stdout,
		FileId:  args.String("fileId"),
		JsonOut: args.Bool("jsonOut"),
	})
	utils.CheckErr(err)
}

func shareRevokeHandler(ctx cli.Context) {
	args := ctx.Args()
	err := utils.NewDrive(args).RevokePermission(drive.RevokePermissionArgs{
		Out:          os.Stdout,
		FileId:       args.String("fileId"),
		PermissionId: args.String("permissionId"),
		JsonOut:      args.Bool("jsonOut"),
	})
	utils.CheckErr(err)
}

func deleteHandler(ctx cli.Context) {
	args := ctx.Args()
	err := utils.NewDrive(args).Delete(drive.DeleteArgs{
		Out:       os.Stdout,
		Id:        args.String("fileId"),
		Recursive: args.Bool("recursive"),
		JsonOut:   args.Bool("jsonOut"),
	})
	utils.CheckErr(err)
}

func listSyncHandler(ctx cli.Context) {
	args := ctx.Args()
	err := utils.NewDrive(args).ListSync(drive.ListSyncArgs{
		Out:        os.Stdout,
		SkipHeader: args.Bool("skipHeader"),
		JsonOut:    args.Bool("jsonOut"),
	})
	utils.CheckErr(err)
}

func listRecursiveSyncHandler(ctx cli.Context) {
	args := ctx.Args()
	err := utils.NewDrive(args).ListRecursiveSync(drive.ListRecursiveSyncArgs{
		Out:         os.Stdout,
		RootId:      args.String("fileId"),
		SkipHeader:  args.Bool("skipHeader"),
		PathWidth:   args.Int64("pathWidth"),
		SizeInBytes: args.Bool("sizeInBytes"),
		SortOrder:   args.String("sortOrder"),
		JsonOut:     args.Bool("jsonOut"),
	})
	utils.CheckErr(err)
}

func deleteRevisionHandler(ctx cli.Context) {
	args := ctx.Args()
	err := utils.NewDrive(args).DeleteRevision(drive.DeleteRevisionArgs{
		Out:        os.Stdout,
		FileId:     args.String("fileId"),
		RevisionId: args.String("revId"),
		JsonOut:    args.Bool("jsonOut"),
	})
	utils.CheckErr(err)
}

func aboutHandler(ctx cli.Context) {
	args := ctx.Args()
	err := utils.NewDrive(args).About(drive.AboutArgs{
		Out:         os.Stdout,
		SizeInBytes: args.Bool("sizeInBytes"),
		JsonOut:     args.Bool("jsonOut"),
	})
	utils.CheckErr(err)
}

func aboutImportHandler(ctx cli.Context) {
	args := ctx.Args()
	err := utils.NewDrive(args).AboutImport(drive.AboutImportArgs{
		Out: os.Stdout,
	})
	utils.CheckErr(err)
}

func aboutExportHandler(ctx cli.Context) {
	args := ctx.Args()
	err := utils.NewDrive(args).AboutExport(drive.AboutExportArgs{
		Out: os.Stdout,
	})
	utils.CheckErr(err)
}

func progressWriter(discard bool) io.Writer {
	if discard {
		return ioutil.Discard
	}
	return os.Stderr
}

func durationInSeconds(seconds int64) time.Duration {
	return time.Second * time.Duration(seconds)
}

func conflictResolution(args cli.Arguments) drive.ConflictResolution {
	keepLocal := args.Bool("keepLocal")
	keepRemote := args.Bool("keepRemote")
	keepLargest := args.Bool("keepLargest")

	if (keepLocal && keepRemote) || (keepLocal && keepLargest) || (keepRemote && keepLargest) {
		utils.ExitF("Only one conflict resolution flag can be given")
	}

	if keepLocal {
		return drive.KeepLocal
	}

	if keepRemote {
		return drive.KeepRemote
	}

	if keepLargest {
		return drive.KeepLargest
	}

	return drive.NoResolution
}

func checkUploadArgs(args cli.Arguments) {
	if args.Bool("recursive") && args.Bool("delete") {
		utils.ExitF("--delete is not allowed for recursive uploads")
	}

	if args.Bool("recursive") && args.Bool("share") {
		utils.ExitF("--share is not allowed for recursive uploads")
	}
}

func checkDownloadArgs(args cli.Arguments) {
	if args.Bool("recursive") && args.Bool("delete") {
		utils.ExitF("--delete is not allowed for recursive downloads")
	}
}
