resource "local_file" "sqs_queue" {
    for_each = toset([for num in var.prenv_pull_request_numbers : format("%d", num)])
    content = "sqs_queue of pr ${each.value}"
    filename = "${path.module}/queue_${each.value}"
}
