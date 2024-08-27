import {cn} from "@/lib/utils"
import {
    ContextMenu,
    ContextMenuContent,
    ContextMenuItem,
    ContextMenuSeparator,
    ContextMenuTrigger,
} from "@/components/ui/context-menu"
import {FileTypeIcon} from "lucide-react";
import {Card} from "@/components/ui/card";

export function FileView({file}: any) {
    return (
        <div className={cn("space-y-3")}>
            <ContextMenu>
                <ContextMenuTrigger>
                    <Card
                        key={file.id}
                        className={`p-4 flex flex-col gap-4 ${file.selected ? "bg-primary text-primary-foreground" : ""}`}
                    >
                        <div className="flex items-center gap-4">
                            <div
                                className={`bg-muted rounded-md flex items-center justify-center aspect-square w-12 ${
                                    file.selected ? "bg-primary-foreground text-primary" : ""
                                }`}
                            >
                                <FileTypeIcon className="w-6 h-6"/>
                            </div>
                            <div className="flex-1">
                                <h3 className={`font-medium truncate ${file.selected ? "text-primary-foreground" : ""}`}>
                                    {file.name}
                                </h3>
                                <p className={`text-sm ${file.selected ? "text-primary-foreground" : "text-muted-foreground"}`}>
                                    {file.size}
                                </p>
                            </div>
                        </div>
                        <div
                            className={`text-sm ${file.selected ? "text-primary-foreground" : "text-muted-foreground"}`}>
                            Modified: {file.modified}
                        </div>
                    </Card>
                </ContextMenuTrigger>
                <ContextMenuContent className="w-40">
                    <ContextMenuItem>Play Next</ContextMenuItem>
                    <ContextMenuItem>Play Later</ContextMenuItem>
                    <ContextMenuItem>Create Station</ContextMenuItem>
                    <ContextMenuSeparator/>
                    <ContextMenuItem>Like</ContextMenuItem>
                    <ContextMenuItem>Share</ContextMenuItem>
                </ContextMenuContent>
            </ContextMenu>
        </div>
    )
}