package drive

import (
	"encoding/json"
	"fmt"
	"io"
	"text/tabwriter"

	"google.golang.org/api/drive/v3"
)

type ShareArgs struct {
	Out          io.Writer
	FileId       string
	Role         string
	Type         string
	Email        string
	Domain       string
	Discoverable bool
	JsonOut      bool
}

func (g *Drive) Share(args ShareArgs) error {
	permission := &drive.Permission{
		AllowFileDiscovery: args.Discoverable,
		Role:               args.Role,
		Type:               args.Type,
		EmailAddress:       args.Email,
		Domain:             args.Domain,
	}

	p, err := g.service.Permissions.Create(args.FileId, permission).Do()
	if err != nil {
		return fmt.Errorf("failed to share file: %s", err)
	}

	if args.JsonOut {
		if jb, err := json.Marshal(p); err != nil {
			return err
		} else {
			_, _ = fmt.Fprintln(args.Out, string(jb))
			return nil
		}
	}

	_, _ = fmt.Fprintf(args.Out, "Granted %s permission to %s\n", args.Role, args.Type)
	return nil
}

type RevokePermissionArgs struct {
	Out          io.Writer
	FileId       string
	PermissionId string
	JsonOut      bool
}

func (g *Drive) RevokePermission(args RevokePermissionArgs) error {
	err := g.service.Permissions.Delete(args.FileId, args.PermissionId).Do()
	if err != nil {
		return fmt.Errorf("failed to revoke permission: %s", err)
	}

	mesg := "permission revoked"
	if args.JsonOut {
		if jb, err := json.Marshal(map[string]string{
			"mesg": mesg,
		}); err != nil {
			return err
		} else {
			_, _ = fmt.Fprintln(args.Out, string(jb))
			return nil
		}
	}

	_, _ = fmt.Fprintf(args.Out, "%s\n", mesg)
	return nil
}

type ListPermissionsArgs struct {
	Out     io.Writer
	FileId  string
	JsonOut bool
}

func (g *Drive) ListPermissions(args ListPermissionsArgs) error {
	permList, err := g.service.Permissions.List(args.FileId).Fields("permissions(id,role,type,domain,emailAddress,allowFileDiscovery)").Do()
	if err != nil {
		return fmt.Errorf("failed to list permissions: %s", err)
	}

	if args.JsonOut {
		return printPermissionsJson(printPermissionsArgs{
			out:         args.Out,
			permissions: permList.Permissions,
		})
	}

	printPermissions(printPermissionsArgs{
		out:         args.Out,
		permissions: permList.Permissions,
	})
	return nil
}

func (g *Drive) shareAnyoneReader(fileId string) error {
	permission := &drive.Permission{
		Role: "reader",
		Type: "anyone",
	}

	_, err := g.service.Permissions.Create(fileId, permission).Do()
	if err != nil {
		return fmt.Errorf("failed to share file: %s", err)
	}

	return nil
}

type printPermissionsArgs struct {
	out         io.Writer
	permissions []*drive.Permission
}

func printPermissions(args printPermissionsArgs) {
	w := new(tabwriter.Writer)
	w.Init(args.out, 0, 0, 3, ' ', 0)

	_, _ = fmt.Fprintln(w, "Id\tType\tRole\tEmail\tDomain\tDiscoverable")

	for _, p := range args.permissions {
		_, _ = fmt.Fprintf(w, "%s\t%s\t%s\t%s\t%s\t%s\n",
			p.Id,
			p.Type,
			p.Role,
			p.EmailAddress,
			p.Domain,
			formatBool(p.AllowFileDiscovery),
		)
	}

	_ = w.Flush()
}

func printPermissionsJson(args printPermissionsArgs) error {
	jb, err := json.Marshal(args.permissions)
	if err != nil {
		return err
	}

	_, _ = fmt.Fprintln(args.out, string(jb))
	return nil
}
